/*
Copyright © 2025 Denys Makeienko <denys.makeienko@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func uninstallBinary(flv, version string) error {
	target := filepath.Join(versionsDir(flv), version)

	// Defence in depth before RemoveAll: flavor and version are validated
	// upstream, but never delete anything outside ~/.tfenvgo/versions/<flavor>.
	rel, err := filepath.Rel(filepath.Join(rootURL, "versions"), target)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) || !strings.Contains(rel, string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove %s: outside the versions directory", target)
	}

	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s version v%s is not installed", flv, version)
		}
		return fmt.Errorf("failed to check %s version v%s: %w", flv, version, err)
	}

	active, _ := getCurrentVersion(flv)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to remove version %s: %w", version, err)
	}
	LogInfo("Uninstalled %s version v%s", flv, version)

	// Don't leave the active-version symlink pointing at a deleted binary.
	if active == version {
		link := currentSymlinkPath(flv)
		if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("uninstalled the active version but failed to remove %s: %w", link, err)
		}
		LogWarn("v%s was the active %s version; removed %s. Run \"tfenvgo use\" to pick another.", version, flv, link)
	}
	return nil
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall VERSION [REGEX]",
	Short: "Uninstall a specific Terraform or OpenTofu version",
	Long: `Remove an installed Terraform or OpenTofu version from ~/.tfenvgo/versions.
If it is the active version, the ~/.tfenvgo/bin symlink is removed too.

VERSION can be one of:
  x.y.z          an exact installed version
  latest         the newest installed stable version
  latest REGEX   the newest installed version matching the Go regular expression REGEX

Add --include-prerelease to let latest and latest REGEX pick pre-releases.
Without VERSION, TFENVGO_VERSION or TFENVGO_TERRAFORM_VERSION is used if set.`,
	Example: `  tfenvgo uninstall 1.5.7
  tfenvgo uninstall latest '^1\.4\.'
  tfenvgo uninstall 1.8.0 --flavor tofu`,
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completeVersions([]string{latestArg}, true),
	RunE: func(cmd *cobra.Command, args []string) error {
		flv, err := resolveFlavor(flavorFlag)
		if err != nil {
			return err
		}

		fromEnv := getEnv(versionEnvKey, getEnv(terraformVersionEnvKey, ""))
		if len(args) == 0 && fromEnv == "" {
			// Uninstalling is destructive, so never default to a version.
			return errors.New(`specify the version to uninstall, e.g. "tfenvgo uninstall 1.5.7" or "tfenvgo uninstall latest"`)
		}
		requested, pattern, err := parseVersionArgs(args, func() string { return fromEnv })
		if err != nil {
			return err
		}
		version, err := resolveVersion(flv, requested, pattern, targetLocal, []string{latestArg}, PreReleaseVersionsIncluded)
		if err != nil {
			return err
		}
		return uninstallBinary(flv, version)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

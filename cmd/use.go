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

	"github.com/spf13/cobra"
)

func useVersion(flv, version string) error {
	if err := initConfig(); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	binPath := installedBinaryPath(flv, version)
	if _, err := os.Stat(binPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error checking %s path: %w", flv, err)
		}
		LogWarn("%s v%s is not installed", flv, version)
		LogInfo("Trying to install %s v%s", flv, version)
		// Stop here on failure so the symlink never points at a missing binary.
		if err := installBinary(flv, version); err != nil {
			return err
		}
	}

	symlinkPath := currentSymlinkPath(flv)

	// Ensure bin directory exists
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o750); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(binPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Set executable permissions on the symlink target
	// Intentionally allow executable bit for the binary. Permissions are set to 0755.
	if err := os.Chmod(binPath, 0o755); err != nil { // #nosec G302 - the binary must be executable
		return fmt.Errorf("failed to update permissions: %w", err)
	}

	LogInfo("Changed current %s version to v%s", flv, version)
	return nil
}

var useCmd = &cobra.Command{
	Use:   "use [VERSION [REGEX]]",
	Short: "Change the current Terraform or OpenTofu version",
	Long: `Point ~/.tfenvgo/bin/terraform (or ~/.tfenvgo/bin/tofu) at the given version,
installing it first if needed. Keywords such as latest are resolved against the
upstream releases, not only the installed versions.

` + versionArgHelp,
	Example: `  tfenvgo use 1.5.7
  tfenvgo use latest
  tfenvgo use latest '^1\.5\.'
  tfenvgo use latest-allowed
  tfenvgo use 1.8.0 --flavor tofu`,
	Args:              cobra.MaximumNArgs(2),
	ValidArgsFunction: completeVersions([]string{latestArg, latestAllowedArg, minRequiredArg}, true),
	RunE: func(cmd *cobra.Command, args []string) error {
		flv, err := resolveFlavor(flavorFlag)
		if err != nil {
			return err
		}

		requested, pattern, err := parseVersionArgs(args, versionFromEnvOrFile)
		if err != nil {
			return err
		}
		keywords := []string{latestArg, latestAllowedArg, minRequiredArg}
		version, err := resolveVersion(flv, requested, pattern, targetRemote, keywords, PreReleaseVersionsIncluded)
		if err != nil {
			return err
		}
		return useVersion(flv, version)
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

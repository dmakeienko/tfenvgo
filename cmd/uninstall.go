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

func uninstallBinary(flv, version string) error {
	target := filepath.Join(versionsDir(flv), version)
	if _, err := os.Stat(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s version v%s is not installed", flv, version)
		}
		return fmt.Errorf("failed to check %s version v%s: %w", flv, version, err)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to remove version %s: %w", version, err)
	}
	LogInfo("Uninstalled %s version v%s", flv, version)
	return nil
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall a specific Terraform or OpenTofu version",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		flv := resolveFlavor(flavorFlag)

		requested, pattern, err := parseVersionArgs(args, func() string {
			return getEnv(versionEnvKey, getEnv(terraformVersionEnvKey, latestArg))
		})
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

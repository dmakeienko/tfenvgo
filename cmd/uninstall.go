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
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

func uninstallTerraform(version string) {
	os.RemoveAll(filepath.Join(terraformVersionPath, version))
	fmt.Println(Yellow + "Uninstalled Terraform version v" + version + Reset)
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall a specific Terraform version",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var version string
		var versionRegex *regexp.Regexp
		if len(args) == 0 {
			version = getEnv(terraformVersionEnvKey, latestArg)
			if version == "" {
				version = latestArg
			}
		} else if len(args) == 1 {
			version = args[0]
		} else if len(args) == 2 && args[0] == latestArg {
			version = args[0]
			versionRegex = regexp.MustCompile(args[1])
		}

		allowedVersions := map[string]bool{
			latestArg: true,
		}

		if validateArg(version, allowedVersions) != nil {
			return
		}

		if version == latestArg && versionRegex == nil {
			versions, err := getLocalTerraformVersions(PreReleaseVersionsIncluded)
			if err != nil {
				fmt.Println("failed to get latest version: %w", err)
			}
			version = versions[0]
		} else if version == latestArg && versionRegex != nil {
			latestRegexVersion, err := getLatestAllowed("local", versionRegex.String())
			if err != nil {
				fmt.Println(Red + "Failed to get latest regex version: " + err.Error() + Reset)
				return
			}
			version = latestRegexVersion
		}
		uninstallTerraform(version)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

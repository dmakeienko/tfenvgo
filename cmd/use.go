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

func useVersion(version string) {
	err := initConfig()
	if err != nil {
		fmt.Println("failed to create config: %w", err)
	}

	terraformSelectedPath := filepath.Join(terraformVersionPath, version, "terraform")
	if _, err := os.Stat(terraformSelectedPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println(Yellow + "Terraform v" + version + " is not installed" + Reset)
			fmt.Println(Yellow + "Trying to install terraform v" + version + Reset)
			installTerraform(version)
		}
	}
	if _, err := os.Lstat(currentTerraformVersionPath); err == nil {
		os.Remove(currentTerraformVersionPath)
		if err := os.Symlink(terraformSelectedPath, currentTerraformVersionPath); err != nil {
			fmt.Println(Red + "Failed to create symlink: " + err.Error() + Reset)
			return
		}
		if err := os.Chmod(currentTerraformVersionPath, 0775); err != nil {
			fmt.Println(Red + "Failed  update permissions: " + err.Error() + Reset)
			return
		}
	} else {
		if err := os.Symlink(terraformSelectedPath, currentTerraformVersionPath); err != nil {
			fmt.Println(Red + "Failed to create symlink: " + err.Error() + Reset)
			return
		}
		if err := os.Chmod(currentTerraformVersionPath, 0775); err != nil {
			fmt.Println(Red + "Failed  update permissions: " + err.Error() + Reset)
			return
		}
	}
	fmt.Println(Green + "Changed current terraform version to v" + version + Reset)
}

var useCmd = &cobra.Command{
	Use:   "use",
	Short: "Change the current Terraform version",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var version string
		var versionRegex *regexp.Regexp
		versionFromFile, _ := readVersionFromFile()
		if len(args) == 0 {
			version = getEnv(terraformVersionEnvKey, versionFromFile)
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
			latestArg:        true,
			latestAllowedArg: true,
			minRequiredArg:   true,
		}

		if validateArg(version, allowedVersions) != nil {
			return
		}

		switch {
		case (version == latestArg && versionRegex == nil):
			versions, err := getRemoteTerraformVersions(PreReleaseVersionsIncluded)
			if err != nil {
				fmt.Println("failed to use check installed version: %w", err)
				return
			}
			version = versions[0]
		case (version == minRequiredArg):
			minRequiredVersion, err := getMinRequired("remote")
			if err != nil {
				fmt.Println(Red + "Failed to use minimum required version: " + err.Error() + Reset)
				return
			}
			version = minRequiredVersion
		case (version == latestAllowedArg):
			latestAllowedVersion, err := getLatestAllowed("remote", "")
			if err != nil {
				fmt.Println(Red + "Failed to use latest allowed version: " + err.Error() + Reset)
				return
			}
			version = latestAllowedVersion
		case (version == latestArg && versionRegex != nil):
			latestRegexVersion, err := getLatestAllowed("remote", versionRegex.String())
			if err != nil {
				fmt.Println(Red + "Failed to get latest regex version: " + err.Error() + Reset)
				return
			}
			version = latestRegexVersion
		}
		useVersion(version)
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

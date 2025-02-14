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

	"github.com/spf13/cobra"
)

func useVersion(version string) {
	// check if .tfenvgo/bin/terraform exists
	terraformPath := terraformBinPath + "/terraform"
	terraformSelectedPath := terraformVersionPath + "/" + version + "/terraform"
	if _, err := os.Stat(terraformSelectedPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println(Yellow + "Terraform v" + version + " is not installed" + Reset)
			fmt.Println(Yellow + "Use " + "tfenvgo install " + version + " to install it" + Reset)
			return
		}
	}
	if _, err := os.Lstat(terraformPath); err == nil {
		os.Remove(terraformPath)
		if err := os.Symlink(terraformSelectedPath, terraformPath); err != nil {
			fmt.Println(Red + "Failed to create symlink: " + err.Error() + Reset)
			return
		}
		if err := os.Chmod(terraformPath, 0775); err != nil {
			fmt.Println(Red + "Failed  update permissions: " + err.Error() + Reset)
			return
		}
	} else {
		if err := os.Symlink(terraformSelectedPath, terraformPath); err != nil {
			fmt.Println(Red + "Failed to create symlink: " + err.Error() + Reset)
			return
		}
		if err := os.Chmod(terraformPath, 0775); err != nil {
			fmt.Println(Red + "Failed  update permissions: " + err.Error() + Reset)
			return
		}
	}
	fmt.Println(Green + "Changed current terraform version to v" + version + Reset)
}

var useCmd = &cobra.Command{
	Use:   "use",
	Short: "Change the current Terraform version",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := getEnv(terraformVersionEnv, "latest")
		if len(args) > 0 {
			version = args[0]
		}

		allowedVersions := map[string]bool{
			latestArg:        true,
			latestAllowedArg: true,
			minRequiredArg:   true,
		}

		if validateArg(version, allowedVersions) != nil {
			return
		}

		switch version {
		case latestArg:
			versions, err := getTerraformVersions()
			if err != nil {
				fmt.Println("failed to get latest version: %w", err)
			}
			version = versions[0]
		case minRequiredArg:
			version, _ = getMinRequired()
		case latestAllowedArg:
			version, _ = getLatestAllowed()
		}
		useVersion(version)
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}

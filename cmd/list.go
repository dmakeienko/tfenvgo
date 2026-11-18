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
	"regexp"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

func getLocalVersions(flv string, preReleaseVersionsIncluded bool) ([]string, error) {
	files, err := os.ReadDir(versionsDir(flv))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil // nothing installed yet for this flavor
	}
	if err != nil {
		return nil, err
	}

	var versions []*semver.Version
	var versionRegex *regexp.Regexp
	stableVersionRegex := regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)
	preReleaseVersionRegex := regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-z]+\d+)?$`)

	if preReleaseVersionsIncluded {
		versionRegex = preReleaseVersionRegex
	} else {
		versionRegex = stableVersionRegex
	}

	for _, f := range files {
		name := f.Name()
		if versionRegex.MatchString(name) {
			v, err := semver.NewVersion(name)
			if err == nil {
				versions = append(versions, v)
			}
		}
	}

	sort.Sort(sort.Reverse(semver.Collection(versions))) // Sort in descending order, top one is always the latest

	var versionStrings []string
	for _, v := range versions {
		versionStrings = append(versionStrings, v.String())
	}

	return versionStrings, nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed versions",
	Long: `List the installed versions of the current flavor, newest first.
The active version (the one ~/.tfenvgo/bin points to) is marked with --->.`,
	Example: `  tfenvgo list
  tfenvgo list --include-prerelease
  tfenvgo list --flavor tofu`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		flv, err := resolveFlavor(flavorFlag)
		if err != nil {
			return err
		}
		versions, err := getLocalVersions(flv, PreReleaseVersionsIncluded)
		if err != nil {
			return fmt.Errorf("failed to list installed versions: %w", err)
		}
		if len(versions) == 0 {
			fmt.Printf("No %s versions installed. Install one with \"tfenvgo install\".\n", flv)
			return nil
		}
		currentVersion, err := getCurrentVersion(flv)
		if err != nil {
			currentVersion = ""
		}

		fmt.Println(Green + "Installed " + flv + " versions:" + Reset)
		for _, v := range versions {
			if v == currentVersion {
				fmt.Println(Green + "---> " + v + " (set by " + currentSymlinkPath(flv) + ")" + Reset)
			} else {
				fmt.Println("     " + Gray + v + Reset)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

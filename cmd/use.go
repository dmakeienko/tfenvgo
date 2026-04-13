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
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

func useVersion(flv, version string) {
	err := initConfig()
	if err != nil {
		LogError("Failed to create config: %v", err)
		return
	}

	binPath := installedBinaryPath(flv, version)
	if _, err := os.Stat(binPath); err != nil {
		if os.IsNotExist(err) {
			LogWarn("%s v%s is not installed", flv, version)
			LogInfo("Trying to install %s v%s", flv, version)
			installBinary(flv, version)
		} else {
			LogError("Error checking %s path: %v", flv, err)
			return
		}
	}

	symlinkPath := currentSymlinkPath(flv)

	// Ensure bin directory exists
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o750); err != nil {
		LogError("Failed to create bin directory: %v", err)
		return
	}

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			LogError("Failed to remove existing symlink: %v", err)
			return
		}
	}

	// Create new symlink
	if err := os.Symlink(binPath, symlinkPath); err != nil {
		LogError("Failed to create symlink: %v", err)
		return
	}

	// Set executable permissions on the symlink target
	// Intentionally allow executable bit for the binary. Permissions are set to 0755.
	if err := os.Chmod(binPath, 0o755); err != nil {
		LogError("Failed to update permissions: %v", err)
		return
	}

	LogInfo("Changed current %s version to v%s", flv, version)
}

var useCmd = &cobra.Command{
	Use:   "use",
	Short: "Change the current Terraform or OpenTofu version",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		flv := resolveFlavor(flavorFlag)

		var version string
		var versionRegex *regexp.Regexp
		versionFromFile, _ := readVersionFromFile()
		if len(args) == 0 {
			version = getEnv(versionEnvKey, getEnv(terraformVersionEnvKey, versionFromFile))
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
			versions, err := getRemoteVersions(flv, PreReleaseVersionsIncluded)
			if err != nil {
				LogError("failed to use check installed version: %v", err)
				return
			}
			if len(versions) == 0 {
				LogError("no remote versions available")
				return
			}
			version = versions[0]
		case (version == minRequiredArg):
			minRequiredVersion, err := getMinRequired(flv, "remote")
			if err != nil {
				LogError("Failed to use minimum required version: %v", err)
				return
			}
			version = minRequiredVersion
		case (version == latestAllowedArg):
			latestAllowedVersion, err := getLatestAllowed(flv, "remote", "")
			if err != nil {
				LogError("Failed to use latest allowed version: %v", err)
				return
			}
			version = latestAllowedVersion
		case (version == latestArg && versionRegex != nil):
			latestRegexVersion, err := getLatestAllowed(flv, "remote", versionRegex.String())
			if err != nil {
				LogError("Failed to get latest regex version: %v", err)
				return
			}
			version = latestRegexVersion
		}
		useVersion(flv, version)
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

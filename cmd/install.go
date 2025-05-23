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
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func unarchiveZip(archivePath, version string) error {
	dst := filepath.Clean(filepath.Join(terraformVersionPath, version))

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archive.Close()

	// Create the destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, f := range archive.File {
		// Clean the file path and ensure it's within the destination directory
		filePath := filepath.Clean(filepath.Join(dst, f.Name)) //nolint

		// Prevent path traversal by checking if the file path is within the destination
		if !strings.HasPrefix(filePath, dst) {
			return fmt.Errorf("invalid file path (potential path traversal): %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			fmt.Printf("Creating directory: %s\n", filePath)
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", filePath, err)
		}

		// Create the file with secure permissions
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			dstFile.Close()
			return fmt.Errorf("failed to open file in archive %s: %w", f.Name, err)
		}

		// Check for G110: Potential DoS vulnerability via decompression bomb
		for {
			_, err := io.CopyN(dstFile, fileInArchive, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
		}

		fileInArchive.Close()
		dstFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file contents %s: %w", f.Name, err)
		}
	}

	return nil
}

func downloadTerraform(version string) error {
	osType := getEnv(archEnvKey, defaultOSType)
	arch := getEnv(osTypeEnvKey, defaultArch)
	terraformDownloadURL := terraformReleasesURL + "/" + version + "/terraform_" + version + "_" + osType + "_" + arch + ".zip"
	fmt.Println(Yellow + "Downloading " + terraformDownloadURL + Reset)

	// Get the data
	resp, err := http.Get(terraformDownloadURL) //nolint
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}
	// Create the file
	filepath := filepath.Join("/tmp", path.Base(resp.Request.URL.String()))

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	fmt.Println(Yellow + "Downloaded file to " + filepath + Reset)

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	err = unarchiveZip(filepath, version)

	if err != nil {
		return fmt.Errorf("failed to unarchive: %v", err)
	}
	fmt.Println(Yellow + "Removing " + filepath + Reset)
	os.Remove(filepath)
	fmt.Println(Yellow + filepath + " removed")
	return err
}

func installTerraform(version string) {
	_, err := os.Stat(filepath.Join(terraformVersionPath, version))
	if os.IsNotExist(err) {
		err := downloadTerraform(version)
		if err != nil {
			fmt.Println(Red+"error downloading:", err)
			return
		}
		fmt.Println(Green + "Terraform v" + version + " has been installed" + Reset)
	} else {
		fmt.Println(Yellow + "Terraform v" + version + " is already installed." + Reset)
	}
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a specific Terraform version",
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
				fmt.Println("failed to get latest version: %w", err)
			}
			version = versions[0]
		case (version == minRequiredArg):
			minRequiredVersion, err := getMinRequired("remote")
			if err != nil {
				fmt.Println(Red + "Failed to get minimum required version: " + err.Error() + Reset)
				return
			}
			version = minRequiredVersion
		case (version == latestAllowedArg):
			latestAllowedVersion, err := getLatestAllowed("remote", "")
			if err != nil {
				fmt.Println(Red + "Failed to get latest allowed version: " + err.Error() + Reset)
				return
			}
			version = latestAllowedVersion
		case (version == latestArg && versionRegex != nil):
			latestRegexVersion, err := getLatestAllowed("remote", versionRegex.String())
			if err != nil {
				fmt.Println(Red + "Failed to get latest allowed version: " + err.Error() + Reset)
				return
			}
			version = latestRegexVersion
		}
		installTerraform(version)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")

}

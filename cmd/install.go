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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

	// Safety limits to prevent zip bombs
	const (
		maxFiles     = 100
		maxFileSize  = 100 * 1024 * 1024 // 100MB per file
		maxTotalSize = 500 * 1024 * 1024 // 500MB total
		maxDepth     = 2
	)

	var (
		fileCount int
		totalSize int64
	)

	for _, f := range archive.File {
		fileCount++
		if fileCount > maxFiles {
			return fmt.Errorf("too many files in archive (limit: %d)", maxFiles)
		}

		// Clean the file path and ensure it's within the destination directory
		filePath := filepath.Clean(filepath.Join(dst, f.Name))

		// Prevent path traversal by checking if the file path is within the destination
		if !strings.HasPrefix(filePath, dst+string(os.PathSeparator)) && filePath != dst {
			return fmt.Errorf("invalid file path (potential path traversal): %s", f.Name)
		}

		// Check directory depth
		relPath, err := filepath.Rel(dst, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		if strings.Count(relPath, string(os.PathSeparator)) > maxDepth {
			return fmt.Errorf("file path too deep: %s", f.Name)
		}

		// Check file size
		if f.UncompressedSize64 > maxFileSize {
			return fmt.Errorf("file too large: %s (%d bytes)", f.Name, f.UncompressedSize64)
		}

		totalSize += int64(f.UncompressedSize64)
		if totalSize > maxTotalSize {
			return fmt.Errorf("total uncompressed size exceeds limit (%d bytes)", maxTotalSize)
		}

		if f.FileInfo().IsDir() {
			LogInfo("Creating directory: %s", filePath)
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

		// Copy with size limit to prevent decompression bombs
		written, err := io.CopyN(dstFile, fileInArchive, maxFileSize)
		fileInArchive.Close()
		dstFile.Close()

		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to copy file contents %s: %w", f.Name, err)
		}

		// Verify the copied size matches expected
		if written != int64(f.UncompressedSize64) && err != io.EOF {
			return fmt.Errorf("file size mismatch for %s: expected %d, got %d", f.Name, f.UncompressedSize64, written)
		}
	}

	return nil
}

func downloadTerraform(version string) error {
	osType := getEnv(archEnvKey, defaultOSType)
	arch := getEnv(osTypeEnvKey, defaultArch)
	terraformDownloadURL := terraformReleasesURL + "/" + version + "/terraform_" + version + "_" + osType + "_" + arch + ".zip"
	LogInfo("Downloading %s", terraformDownloadURL)

	// Create HTTP client with security configurations
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Create request with context for timeout control
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", terraformDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", "tfenvgo/"+Version)

	// Get the data
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Create the file with secure temp directory
	tempDir := os.TempDir()
	filepath := filepath.Join(tempDir, path.Base(resp.Request.URL.String()))

	out, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()
	LogInfo("Downloaded file to %s", filepath)

	// Write the body to file with size limit to prevent zip bombs
	const maxFileSize = 500 * 1024 * 1024 // 500MB limit
	_, err = io.CopyN(out, resp.Body, maxFileSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to write file: %w", err)
	}

	err = unarchiveZip(filepath, version)
	if err != nil {
		return fmt.Errorf("failed to unarchive: %w", err)
	}

	LogInfo("Removing %s", filepath)
	if err := os.Remove(filepath); err != nil {
		LogWarn("Warning: failed to remove temp file: %s", err.Error())
	}
	LogInfo("%s removed", filepath)
	return nil
}

func installTerraform(version string) {
	_, err := os.Stat(filepath.Join(terraformVersionPath, version))
	if os.IsNotExist(err) {
		err := downloadTerraform(version)
		if err != nil {
			LogError("error downloading: %v", err)
			return
		}
		LogInfo("Terraform v%s has been installed", version)
	} else {
		LogWarn("Terraform v%s is already installed.", version)
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

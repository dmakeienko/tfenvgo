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
	"math"
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
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Safety limits to prevent zip bombs
	const (
		maxFiles            = 100
		maxFileSize  int64  = 100 * 1024 * 1024 // 100MB per file
		maxTotalSize uint64 = 500 * 1024 * 1024 // 500MB total
		maxDepth            = 10
	)

	var (
		fileCount int
		totalSize uint64
	)

	for _, f := range archive.File {
		fileCount++
		if fileCount > maxFiles {
			return fmt.Errorf("too many files in archive (limit: %d)", maxFiles)
		}

		// Sanitize the archive filename. Use path (slash-separated) for entries inside zip
		internalPath := path.Clean(f.Name)
		if internalPath == "." || internalPath == "" {
			// skip empty entries
			continue
		}
		// Reject absolute paths and parent traversals
		if path.IsAbs(internalPath) || strings.HasPrefix(internalPath, "../") || strings.Contains(internalPath, "..") {
			return fmt.Errorf("invalid file path (potential path traversal): %s", f.Name)
		}

		// Convert to OS-specific path and join with dst
		filePath := filepath.Join(dst, filepath.FromSlash(internalPath))

		// Ensure the result is still within dst (extra abs check to satisfy scanners)
		relPath, err := filepath.Rel(dst, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		if strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("invalid file path (potential path traversal): %s", f.Name)
		}
		if strings.Count(relPath, string(os.PathSeparator)) > maxDepth {
			return fmt.Errorf("file path too deep: %s", f.Name)
		}
		absDst, _ := filepath.Abs(dst)
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		if !strings.HasPrefix(absFile, absDst+string(os.PathSeparator)) && absFile != absDst {
			return fmt.Errorf("invalid file path (outside destination): %s", f.Name)
		}

		// Check file size and total size safely
		if f.UncompressedSize64 > uint64(maxFileSize) {
			return fmt.Errorf("file too large: %s (%d bytes)", f.Name, f.UncompressedSize64)
		}

		totalSize += f.UncompressedSize64
		if totalSize > maxTotalSize {
			return fmt.Errorf("total uncompressed size exceeds limit (%d bytes)", maxTotalSize)
		}

		if f.FileInfo().IsDir() {
			LogInfo("Creating directory: %s", filePath)
			if err := os.MkdirAll(filePath, 0o750); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", filePath, err)
		}

		// Create the file with secure permissions (start restrictive). Make terraform executable if that's the file.
		perm := os.FileMode(0o600)
		if filepath.Base(filePath) == "terraform" {
			perm = 0o755
		}
		// Path is validated above; safe to open.
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		fileInArchive, err := f.Open()
		if err != nil {
			_ = dstFile.Close()
			return fmt.Errorf("failed to open file in archive %s: %w", f.Name, err)
		}
		// Ensure closes are checked
		defer func() {
			if cerr := fileInArchive.Close(); cerr != nil {
				LogWarn("failed to close archive file %s: %v", f.Name, cerr)
			}
		}()

		// Before copying, ensure expected size fits in int64 for safe comparison
		if f.UncompressedSize64 > uint64(math.MaxInt64) {
			return fmt.Errorf("file too large to process: %s (%d bytes)", f.Name, f.UncompressedSize64)
		}

		// Copy with size limit to prevent decompression bombs
		written, err := io.CopyN(dstFile, fileInArchive, maxFileSize)
		if cerr := dstFile.Close(); cerr != nil {
			LogWarn("failed to close destination file %s: %v", filePath, cerr)
		}

		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to copy file contents %s: %w", f.Name, err)
		}

		// Verify the copied size matches expected (guard written >= 0 first)
		if written < 0 {
			return fmt.Errorf("negative write size for %s: %d", f.Name, written)
		}
		if uint64(written) != f.UncompressedSize64 && err != io.EOF {
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

	// Create a secure temp file
	tempDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tempDir, "tfenvgo-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	filepath := tmpFile.Name()
	// ensure file is closed and removed on errors
	defer func() {
		_ = tmpFile.Close()
	}()
	LogInfo("Downloaded file to %s", filepath)

	// Write the body to file with size limit to prevent zip bombs
	const maxFileSize = 500 * 1024 * 1024 // 500MB limit
	// Write with size cap
	_, err = io.CopyN(tmpFile, resp.Body, maxFileSize)
	if err != nil && err != io.EOF {
		_ = os.Remove(filepath)
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
				LogError("failed to get latest version: %v", err)
				return
			}
			if len(versions) == 0 {
				LogError("no remote versions found")
				return
			}
			version = versions[0]
		case (version == minRequiredArg):
			minRequiredVersion, err := getMinRequired("remote")
			if err != nil {
				LogError("Failed to get minimum required version: %v", err)
				return
			}
			version = minRequiredVersion
		case (version == latestAllowedArg):
			latestAllowedVersion, err := getLatestAllowed("remote", "")
			if err != nil {
				LogError("Failed to get latest allowed version: %v", err)
				return
			}
			version = latestAllowedVersion
		case (version == latestArg && versionRegex != nil):
			latestRegexVersion, err := getLatestAllowed("remote", versionRegex.String())
			if err != nil {
				LogError("Failed to get latest allowed version: %v", err)
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

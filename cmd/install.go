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
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func unarchiveZip(archivePath, flv, version string) error {
	dst := filepath.Clean(versionsDir(flv))
	dst = filepath.Join(dst, version)

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
		maxFileSize  int64  = 200 * 1024 * 1024 // 200MB per file
		maxTotalSize uint64 = 500 * 1024 * 1024 // 500MB total
		maxDepth            = 10
	)

	var (
		fileCount int
		totalSize uint64
	)

	binName := binaryName(flv)

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
			// #nosec G703 - Path is validated above with filepath.Rel and HasPrefix checks
			if err := os.MkdirAll(filePath, 0o750); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directories if they don't exist
		// #nosec G703 - Path is validated above with filepath.Rel and HasPrefix checks
		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", filePath, err)
		}

		// Create the file with secure permissions. Make the binary executable.
		perm := os.FileMode(0o600)
		if filepath.Base(filePath) == binName {
			perm = 0o755
		}
		// #nosec G703 - Path is validated above with filepath.Rel and HasPrefix checks
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

const (
	// maxArchiveSize caps the downloaded archive size.
	maxArchiveSize int64 = 500 * 1024 * 1024 // 500MB
	// maxChecksumFileSize caps the downloaded SHA256SUMS file size.
	maxChecksumFileSize int64 = 1024 * 1024 // 1MB
)

// releaseArtifact returns the archive URL, the SHA256SUMS URL and the archive
// file name (as listed in SHA256SUMS) for a flavor/version/platform.
func releaseArtifact(flv, version, osType, arch string) (archiveURL, sumsURL, archiveName string) {
	switch flv {
	case FlavorTofu:
		base := tofuDownloadBase + "/v" + version + "/"
		archiveName = "tofu_" + version + "_" + osType + "_" + arch + ".zip"
		return base + archiveName, base + "tofu_" + version + "_SHA256SUMS", archiveName
	default:
		base := terraformReleasesURL + "/" + version + "/"
		archiveName = "terraform_" + version + "_" + osType + "_" + arch + ".zip"
		return base + archiveName, base + "terraform_" + version + "_SHA256SUMS", archiveName
	}
}

// newHTTPClient returns an HTTP client with the given timeout and TLS 1.2+.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

// httpGet performs a GET request and returns the response only for 200 OK.
// The caller must close the response body.
func httpGet(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "tfenvgo/"+Version)

	// #nosec G704 - URL is constructed from hardcoded base URLs and version
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp, nil
}

// parseChecksum returns the lowercase hex SHA-256 listed for name in a
// sha256sum-formatted file ("<hex>  <file>" per line).
func parseChecksum(sums []byte, name string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// A leading '*' marks binary mode in sha256sum output.
		if strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		sum := strings.ToLower(fields[0])
		if _, err := hex.DecodeString(sum); err != nil || len(sum) != sha256.Size*2 {
			return "", fmt.Errorf("malformed checksum for %s: %q", name, fields[0])
		}
		return sum, nil
	}
	return "", fmt.Errorf("no checksum for %s in SHA256SUMS", name)
}

// fetchExpectedChecksum downloads SHA256SUMS and returns the checksum for archiveName.
func fetchExpectedChecksum(ctx context.Context, client *http.Client, sumsURL, archiveName string) (string, error) {
	resp, err := httpGet(ctx, client, sumsURL)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumFileSize))
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}
	return parseChecksum(body, archiveName)
}

func downloadBinary(flv, version string) error {
	osType := getEnv(osTypeEnvKey, defaultOSType)
	arch := getEnv(archEnvKey, defaultArch)
	archiveURL, sumsURL, archiveName := releaseArtifact(flv, version, osType, arch)

	timeout := downloadTimeout()
	client := newHTTPClient(timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Fetch the published checksum first: if it is missing or malformed there
	// is no point downloading the archive.
	expected, err := fetchExpectedChecksum(ctx, client, sumsURL, archiveName)
	if err != nil {
		return err
	}

	LogInfo("Downloading %s", archiveURL)
	resp, err := httpGet(ctx, client, archiveURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "tfenvgo-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	// Remove the temp archive on every path, including failures.
	defer func() {
		// #nosec G703 - tmpFilePath is from os.CreateTemp(), safe temporary path
		if err := os.Remove(tmpFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			LogWarn("failed to remove temp file %s: %v", tmpFilePath, err)
		}
	}()

	// Hash while writing. Read one byte past the limit to detect oversized
	// bodies instead of silently truncating them.
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmpFile, hasher), io.LimitReader(resp.Body, maxArchiveSize+1))
	closeErr := tmpFile.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	if written > maxArchiveSize {
		return fmt.Errorf("archive exceeds the %d byte limit", maxArchiveSize)
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", archiveName, expected, actual)
	}
	LogInfo("Verified SHA-256 checksum of %s", archiveName)

	if err := unarchiveZip(tmpFilePath, flv, version); err != nil {
		// Do not leave a partial install behind: it would look installed next time.
		if rmErr := os.RemoveAll(filepath.Join(versionsDir(flv), version)); rmErr != nil {
			LogWarn("failed to clean up partial install: %v", rmErr)
		}
		return fmt.Errorf("failed to unarchive: %w", err)
	}
	return nil
}

func installBinary(flv, version string) error {
	binPath := installedBinaryPath(flv, version)
	_, err := os.Stat(binPath)
	if err == nil {
		LogWarn("%s v%s is already installed.", flv, version)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check %s: %w", binPath, err)
	}

	if err := downloadBinary(flv, version); err != nil {
		return fmt.Errorf("failed to install %s v%s: %w", flv, version, err)
	}
	LogInfo("%s v%s has been installed", flv, version)
	return nil
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a specific version of Terraform or OpenTofu",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		flv := resolveFlavor(flavorFlag)

		requested, pattern, err := parseVersionArgs(args, versionFromEnvOrFile)
		if err != nil {
			return err
		}
		keywords := []string{latestArg, latestAllowedArg, minRequiredArg}
		version, err := resolveVersion(flv, requested, pattern, targetRemote, keywords, PreReleaseVersionsIncluded)
		if err != nil {
			return err
		}
		return installBinary(flv, version)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

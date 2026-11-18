package cmd

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// setupRoot points tfenvgo at a fresh temporary root directory and clears
// the environment variables that influence version and flavor resolution.
func setupRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	oldRoot, oldBin, oldVersions := rootURL, terraformBinPath, terraformVersionPath
	rootURL = dir
	terraformBinPath = filepath.Join(dir, "bin")
	terraformVersionPath = filepath.Join(dir, "versions")
	t.Cleanup(func() {
		rootURL, terraformBinPath, terraformVersionPath = oldRoot, oldBin, oldVersions
	})

	for _, key := range []string{
		flavorEnvKey, versionEnvKey, terraformVersionEnvKey,
		osTypeEnvKey, archEnvKey, downloadTimeoutEnvKey,
	} {
		unsetEnv(t, key)
	}
	return dir
}

// unsetEnv removes key for the duration of the test and restores it afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "") // registers restoration of the original value
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
}

// installFakeVersions creates installed-version directories with a dummy binary.
func installFakeVersions(t *testing.T, flv string, versions ...string) {
	t.Helper()
	for _, v := range versions {
		bin := installedBinaryPath(flv, v)
		if err := os.MkdirAll(filepath.Dir(bin), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(bin, []byte("fake"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// setReleaseURLs points all upstream release URLs at base (an httptest server).
func setReleaseURLs(t *testing.T, base string) {
	t.Helper()
	oldTF, oldAPI, oldDL := terraformReleasesURL, tofuReleasesAPI, tofuDownloadBase
	terraformReleasesURL = base + "/terraform"
	tofuReleasesAPI = base + "/tofu/releases"
	tofuDownloadBase = base + "/tofu/download"
	t.Cleanup(func() {
		terraformReleasesURL, tofuReleasesAPI, tofuDownloadBase = oldTF, oldAPI, oldDL
	})
}

// makeZip builds an in-memory zip archive. Entries are written in order.
func makeZip(t *testing.T, entries ...[2]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s not to exist, stat err = %v", path, err)
	}
}

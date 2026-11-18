package cmd

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
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
		osTypeEnvKey, archEnvKey, downloadTimeoutEnvKey, skipSignatureEnvKey,
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

type testKeyPair struct {
	trusted   *openpgp.Entity // stands in for the vendor release key
	untrusted *openpgp.Entity // any other key, e.g. an attacker's
}

var (
	testKeysOnce sync.Once
	testKeysVal  testKeyPair
	testKeysErr  error
)

// testKeys returns throwaway OpenPGP keys, generated once per test binary.
func testKeys(t *testing.T) testKeyPair {
	t.Helper()
	testKeysOnce.Do(func() {
		cfg := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA}
		testKeysVal.trusted, testKeysErr = openpgp.NewEntity("tfenvgo test release", "", "release@example.invalid", cfg)
		if testKeysErr != nil {
			return
		}
		testKeysVal.untrusted, testKeysErr = openpgp.NewEntity("someone else", "", "other@example.invalid", cfg)
	})
	if testKeysErr != nil {
		t.Fatalf("generating test keys: %v", testKeysErr)
	}
	return testKeysVal
}

// trustTestReleaseKey makes the trusted test key the release key of both
// flavors for the duration of the test.
func trustTestReleaseKey(t *testing.T) {
	t.Helper()
	oldTF, oldTofu := terraformSigningKey, tofuSigningKey
	key := armoredPublicKey(t, testKeys(t).trusted)
	terraformSigningKey, tofuSigningKey = key, key
	t.Cleanup(func() { terraformSigningKey, tofuSigningKey = oldTF, oldTofu })
}

func armoredPublicKey(t *testing.T, e *openpgp.Entity) string {
	t.Helper()
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Serialize(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// signDetached returns a binary detached signature of msg, like the
// .sig/.gpgsig files published next to SHA256SUMS.
func signDetached(t *testing.T, signer *openpgp.Entity, msg []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := openpgp.DetachSign(&buf, signer, bytes.NewReader(msg), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func signDetachedArmored(t *testing.T, signer *openpgp.Entity, msg []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := openpgp.ArmoredDetachSign(&buf, signer, bytes.NewReader(msg), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

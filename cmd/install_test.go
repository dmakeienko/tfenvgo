package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
)

const testArchiveName = "terraform_1.5.7_linux_amd64.zip"

func TestParseChecksum(t *testing.T) {
	good := strings.Repeat("ab", 32)
	tests := []struct {
		name    string
		sums    string
		want    string
		wantErr string
	}{
		{name: "found", sums: good + "  " + testArchiveName + "\n", want: good},
		{name: "among other entries", sums: strings.Repeat("cd", 32) + "  terraform_1.5.7_darwin_arm64.zip\n" + good + "  " + testArchiveName + "\n", want: good},
		{name: "binary-mode marker", sums: good + " *" + testArchiveName + "\n", want: good},
		{name: "uppercase hex", sums: strings.ToUpper(good) + "  " + testArchiveName + "\n", want: good},
		{name: "CRLF line endings", sums: good + "  " + testArchiveName + "\r\n", want: good},
		{name: "missing entry", sums: good + "  terraform_1.5.7_darwin_arm64.zip\n", wantErr: "no checksum for"},
		{name: "short hash", sums: "abcd  " + testArchiveName + "\n", wantErr: "malformed checksum"},
		{name: "non-hex hash", sums: strings.Repeat("zz", 32) + "  " + testArchiveName + "\n", wantErr: "malformed checksum"},
		{name: "empty file", sums: "", wantErr: "no checksum for"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseChecksum([]byte(tt.sums), testArchiveName)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReleaseArtifact(t *testing.T) {
	rel := releaseArtifact(FlavorTerraform, "1.5.7", "linux", "amd64")
	want := releaseFiles{
		archiveURL:  "https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip",
		sumsURL:     "https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_SHA256SUMS",
		sigURL:      "https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_SHA256SUMS.72D7468F.sig",
		archiveName: testArchiveName,
	}
	if rel != want {
		t.Errorf("terraform release files = %+v, want %+v", rel, want)
	}

	rel = releaseArtifact(FlavorTofu, "1.8.0", "darwin", "arm64")
	want = releaseFiles{
		archiveURL:  "https://github.com/opentofu/opentofu/releases/download/v1.8.0/tofu_1.8.0_darwin_arm64.zip",
		sumsURL:     "https://github.com/opentofu/opentofu/releases/download/v1.8.0/tofu_1.8.0_SHA256SUMS",
		sigURL:      "https://github.com/opentofu/opentofu/releases/download/v1.8.0/tofu_1.8.0_SHA256SUMS.gpgsig",
		archiveName: "tofu_1.8.0_darwin_arm64.zip",
	}
	if rel != want {
		t.Errorf("tofu release files = %+v, want %+v", rel, want)
	}
}

// fakeRelease is what serveRelease serves for one linux/amd64 release.
// A nil sums or sig makes that file return 404.
type fakeRelease struct {
	archive []byte
	sums    []byte
	sig     []byte
}

// serveRelease serves rel for flv/version from an httptest server, points
// the release URLs at it and trusts the test release key for both flavors.
// It returns a counter of archive downloads.
func serveRelease(t *testing.T, flv, version string, rel fakeRelease) *atomic.Int32 {
	t.Helper()
	hits := new(atomic.Int32)
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	setReleaseURLs(t, srv.URL)
	trustTestReleaseKey(t)
	t.Setenv(osTypeEnvKey, "linux")
	t.Setenv(archEnvKey, "amd64")

	files := releaseArtifact(flv, version, "linux", "amd64")
	handle := func(url string, body []byte, counter *atomic.Int32) {
		if body == nil {
			return
		}
		mux.HandleFunc(strings.TrimPrefix(url, srv.URL), func(w http.ResponseWriter, r *http.Request) {
			if counter != nil {
				counter.Add(1)
			}
			_, _ = w.Write(body)
		})
	}
	handle(files.archiveURL, rel.archive, hits)
	handle(files.sumsURL, rel.sums, nil)
	handle(files.sigURL, rel.sig, nil)
	return hits
}

// serveTerraformRelease serves a fake terraform 1.5.7 linux/amd64 release
// whose SHA256SUMS is validly signed by the trusted test key. An empty sums
// string makes the SHA256SUMS file and its signature return 404. It returns
// a counter of archive downloads.
func serveTerraformRelease(t *testing.T, archive []byte, sums string) *atomic.Int32 {
	t.Helper()
	rel := fakeRelease{archive: archive}
	if sums != "" {
		rel.sums = []byte(sums)
		rel.sig = signDetached(t, testKeys(t).trusted, rel.sums)
	}
	return serveRelease(t, FlavorTerraform, "1.5.7", rel)
}

func TestInstallVerifiesChecksum(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "#!/bin/sh\necho fake terraform\n"})
	sums := strings.Repeat("00", 32) + "  terraform_1.5.7_darwin_arm64.zip\n" +
		sha256Hex(archive) + "  " + testArchiveName + "\n"
	serveTerraformRelease(t, archive, sums)

	if err := installBinary(FlavorTerraform, "1.5.7"); err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	got, err := os.ReadFile(installedBinaryPath(FlavorTerraform, "1.5.7"))
	if err != nil {
		t.Fatalf("reading installed binary: %v", err)
	}
	if string(got) != "#!/bin/sh\necho fake terraform\n" {
		t.Fatalf("installed binary content = %q", got)
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "tampered"})
	sums := sha256Hex([]byte("something else")) + "  " + testArchiveName + "\n"
	serveTerraformRelease(t, archive, sums)

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestInstallFailsWithoutChecksumFile(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "x"})
	hits := serveTerraformRelease(t, archive, "")

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "failed to download checksums") {
		t.Fatalf("err = %v, want checksum download failure", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("archive downloaded %d times; it should not be fetched without a checksum", n)
	}
	assertNotExist(t, installedBinaryPath(FlavorTerraform, "1.5.7"))
}

func TestInstallFailsWhenArchiveNotListedInChecksums(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "x"})
	serveTerraformRelease(t, archive, sha256Hex(archive)+"  terraform_1.5.7_darwin_arm64.zip\n")

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "no checksum for "+testArchiveName) {
		t.Fatalf("err = %v, want missing checksum entry", err)
	}
}

func TestInstallCleansUpFailedExtraction(t *testing.T) {
	setupRoot(t)
	// Valid checksum, but more entries than unarchiveZip allows: extraction
	// starts, then fails part-way through.
	entries := make([][2]string, 0, 101)
	for i := 0; i < 101; i++ {
		entries = append(entries, [2]string{fmt.Sprintf("f%03d", i), "x"})
	}
	archive := makeZip(t, entries...)
	serveTerraformRelease(t, archive, sha256Hex(archive)+"  "+testArchiveName+"\n")

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "failed to unarchive") {
		t.Fatalf("err = %v, want unarchive failure", err)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestUnarchiveZipRejectsPathTraversal(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"../evil", "pwned"})
	archivePath := filepath.Join(t.TempDir(), "evil.zip")
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := unarchiveZip(archivePath, FlavorTerraform, "1.0.0"); err == nil {
		t.Fatal("expected an error for a path-traversal entry")
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "evil"))
}

// signedTerraformRelease returns a fake terraform 1.5.7 linux/amd64 release
// whose SHA256SUMS lists archive and is signed by signer.
func signedTerraformRelease(t *testing.T, archive []byte, signer *openpgp.Entity) fakeRelease {
	t.Helper()
	sums := []byte(sha256Hex(archive) + "  " + testArchiveName + "\n")
	return fakeRelease{archive: archive, sums: sums, sig: signDetached(t, signer, sums)}
}

func TestInstallVerifiesSignature(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "genuine"})
	serveRelease(t, FlavorTerraform, "1.5.7", signedTerraformRelease(t, archive, testKeys(t).trusted))

	if err := installBinary(FlavorTerraform, "1.5.7"); err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	if _, err := os.Stat(installedBinaryPath(FlavorTerraform, "1.5.7")); err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
}

func TestInstallAcceptsArmoredSignature(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "genuine"})
	rel := signedTerraformRelease(t, archive, testKeys(t).trusted)
	rel.sig = signDetachedArmored(t, testKeys(t).trusted, rel.sums)
	serveRelease(t, FlavorTerraform, "1.5.7", rel)

	if err := installBinary(FlavorTerraform, "1.5.7"); err != nil {
		t.Fatalf("installBinary: %v", err)
	}
}

// A compromised mirror serves its own archive and a SHA256SUMS that matches
// it, but cannot produce a matching signature.
func TestInstallRejectsTamperedChecksums(t *testing.T) {
	setupRoot(t)
	genuine := makeZip(t, [2]string{"terraform", "genuine"})
	evil := makeZip(t, [2]string{"terraform", "evil"})
	rel := signedTerraformRelease(t, genuine, testKeys(t).trusted)
	rel.archive = evil
	rel.sums = []byte(sha256Hex(evil) + "  " + testArchiveName + "\n")
	hits := serveRelease(t, FlavorTerraform, "1.5.7", rel)

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "signature verification of terraform_1.5.7_SHA256SUMS failed") {
		t.Fatalf("err = %v, want signature verification failure", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("archive downloaded %d times; it should not be fetched with an unverified checksum", n)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestInstallRejectsSignatureFromUntrustedKey(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "evil"})
	hits := serveRelease(t, FlavorTerraform, "1.5.7", signedTerraformRelease(t, archive, testKeys(t).untrusted))

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "not made by the trusted release key") {
		t.Fatalf("err = %v, want untrusted key failure", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("archive downloaded %d times", n)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestInstallRejectsGarbageSignature(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "x"})
	rel := signedTerraformRelease(t, archive, testKeys(t).trusted)
	rel.sig = []byte("not a signature")
	serveRelease(t, FlavorTerraform, "1.5.7", rel)

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("err = %v, want invalid signature", err)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestInstallFailsWithoutSignature(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "x"})
	rel := signedTerraformRelease(t, archive, testKeys(t).trusted)
	rel.sig = nil
	hits := serveRelease(t, FlavorTerraform, "1.5.7", rel)

	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "failed to download checksum signature") {
		t.Fatalf("err = %v, want signature download failure", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("archive downloaded %d times", n)
	}
	assertNotExist(t, filepath.Join(versionsDir(FlavorTerraform), "1.5.7"))
}

func TestInstallSkipSignatureEscapeHatch(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"terraform", "unsigned"})
	rel := signedTerraformRelease(t, archive, testKeys(t).trusted)
	rel.sig = nil
	serveRelease(t, FlavorTerraform, "1.5.7", rel)

	for _, v := range []string{"true", "yes", "0"} {
		t.Setenv(skipSignatureEnvKey, v)
		if err := installBinary(FlavorTerraform, "1.5.7"); err == nil {
			t.Fatalf("%s=%s: install succeeded without a signature; only \"1\" may skip it", skipSignatureEnvKey, v)
		}
	}

	t.Setenv(skipSignatureEnvKey, "1")
	if err := installBinary(FlavorTerraform, "1.5.7"); err != nil {
		t.Fatalf("installBinary with %s=1: %v", skipSignatureEnvKey, err)
	}

	// Skipping the signature never skips the checksum.
	setupRoot(t)
	t.Setenv(skipSignatureEnvKey, "1")
	rel.archive = makeZip(t, [2]string{"terraform", "tampered"})
	serveRelease(t, FlavorTerraform, "1.5.7", rel)
	err := installBinary(FlavorTerraform, "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
}

func TestInstallTofuVerifiesSignature(t *testing.T) {
	setupRoot(t)
	archive := makeZip(t, [2]string{"tofu", "genuine tofu"})
	sums := []byte(sha256Hex(archive) + "  tofu_1.8.0_linux_amd64.zip\n")
	rel := fakeRelease{archive: archive, sums: sums, sig: signDetached(t, testKeys(t).trusted, sums)}
	serveRelease(t, FlavorTofu, "1.8.0", rel)
	if err := installBinary(FlavorTofu, "1.8.0"); err != nil {
		t.Fatalf("installBinary: %v", err)
	}

	// tofu releases are checked against the tofu key, not the terraform one.
	setupRoot(t)
	serveRelease(t, FlavorTofu, "1.8.0", rel)
	tofuSigningKey = armoredPublicKey(t, testKeys(t).untrusted) // restored by serveRelease's cleanup
	err := installBinary(FlavorTofu, "1.8.0")
	if err == nil || !strings.Contains(err.Error(), "not made by the trusted release key") {
		t.Fatalf("err = %v, want untrusted key failure", err)
	}
}

// TestEmbeddedKeyFingerprints pins the embedded release keys to the
// fingerprints published by HashiCorp and OpenTofu.
func TestEmbeddedKeyFingerprints(t *testing.T) {
	for name, tc := range map[string]struct{ key, fingerprint string }{
		"hashicorp": {hashicorpPublicKey, hashicorpKeyFingerprint},
		"opentofu":  {opentofuPublicKey, opentofuKeyFingerprint},
	} {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(tc.key))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(keyring) != 1 {
			t.Fatalf("%s: keyring has %d entities, want 1", name, len(keyring))
		}
		if got := fmt.Sprintf("%X", keyring[0].PrimaryKey.Fingerprint); got != tc.fingerprint {
			t.Errorf("%s fingerprint = %s, want %s", name, got, tc.fingerprint)
		}
	}
	if terraformSigningKey != hashicorpPublicKey || tofuSigningKey != opentofuPublicKey {
		t.Error("production signing keys are not the embedded vendor keys")
	}
}

// TestRealReleaseSignatures checks the embedded keys against signatures
// published upstream; testdata holds unmodified copies of the real files.
func TestRealReleaseSignatures(t *testing.T) {
	readTestdata := func(t *testing.T, name string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, tc := range []struct {
		key, fingerprint, sums, sig string
	}{
		{hashicorpPublicKey, hashicorpKeyFingerprint, "terraform_1.5.7_SHA256SUMS", "terraform_1.5.7_SHA256SUMS.72D7468F.sig"},
		{opentofuPublicKey, opentofuKeyFingerprint, "tofu_1.8.0_SHA256SUMS", "tofu_1.8.0_SHA256SUMS.gpgsig"},
	} {
		t.Run(tc.sums, func(t *testing.T) {
			sums, sig := readTestdata(t, tc.sums), readTestdata(t, tc.sig)
			got, err := verifySignature(tc.key, sums, sig)
			if err != nil {
				t.Fatalf("verifySignature: %v", err)
			}
			if got != tc.fingerprint {
				t.Errorf("signer = %s, want %s", got, tc.fingerprint)
			}

			tampered := append([]byte{}, sums...)
			tampered[0] ^= 1
			if _, err := verifySignature(tc.key, tampered, sig); err == nil {
				t.Error("tampered SHA256SUMS verified")
			}
		})
	}

	t.Run("keys are not interchangeable", func(t *testing.T) {
		sums := readTestdata(t, "terraform_1.5.7_SHA256SUMS")
		sig := readTestdata(t, "terraform_1.5.7_SHA256SUMS.72D7468F.sig")
		if _, err := verifySignature(opentofuPublicKey, sums, sig); err == nil {
			t.Error("OpenTofu key accepted a HashiCorp signature")
		}
	})
}

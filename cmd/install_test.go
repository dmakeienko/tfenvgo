package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
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
	archive, sums, name := releaseArtifact(FlavorTerraform, "1.5.7", "linux", "amd64")
	if want := "https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip"; archive != want {
		t.Errorf("terraform archive URL = %q, want %q", archive, want)
	}
	if want := "https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_SHA256SUMS"; sums != want {
		t.Errorf("terraform sums URL = %q, want %q", sums, want)
	}
	if name != testArchiveName {
		t.Errorf("terraform archive name = %q, want %q", name, testArchiveName)
	}

	archive, sums, name = releaseArtifact(FlavorTofu, "1.8.0", "darwin", "arm64")
	if want := "https://github.com/opentofu/opentofu/releases/download/v1.8.0/tofu_1.8.0_darwin_arm64.zip"; archive != want {
		t.Errorf("tofu archive URL = %q, want %q", archive, want)
	}
	if want := "https://github.com/opentofu/opentofu/releases/download/v1.8.0/tofu_1.8.0_SHA256SUMS"; sums != want {
		t.Errorf("tofu sums URL = %q, want %q", sums, want)
	}
	if want := "tofu_1.8.0_darwin_arm64.zip"; name != want {
		t.Errorf("tofu archive name = %q, want %q", name, want)
	}
}

// serveTerraformRelease serves a fake terraform 1.5.7 linux/amd64 release.
// An empty sums string makes the SHA256SUMS file return 404. It returns a
// counter of archive downloads.
func serveTerraformRelease(t *testing.T, archive []byte, sums string) *atomic.Int32 {
	t.Helper()
	hits := new(atomic.Int32)
	mux := http.NewServeMux()
	mux.HandleFunc("/terraform/1.5.7/"+testArchiveName, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(archive)
	})
	if sums != "" {
		mux.HandleFunc("/terraform/1.5.7/terraform_1.5.7_SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, sums)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	setReleaseURLs(t, srv.URL)
	t.Setenv(osTypeEnvKey, "linux")
	t.Setenv(archEnvKey, "amd64")
	return hits
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

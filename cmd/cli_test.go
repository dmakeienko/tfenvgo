package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runCLI executes the root command with args and returns its error, which is
// what decides the process exit code in Execute.
func runCLI(t *testing.T, args ...string) error {
	t.Helper()
	return runCLIWithOutput(t, io.Discard, args...)
}

// runCLIWithOutput is runCLI with the command's output (cmd.OutOrStdout) sent to out.
func runCLIWithOutput(t *testing.T, out io.Writer, args ...string) error {
	t.Helper()
	// Flag variables keep their values between Execute calls; reset them so
	// each run only sees the flags it passes.
	flavorFlag = ""
	PreReleaseVersionsIncluded = false
	rootCmd.SetArgs(args)
	rootCmd.SetOut(out)
	rootCmd.SetErr(io.Discard)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	return rootCmd.Execute()
}

// TestCommandsReturnErrors guards the exit code: every one of these used to
// log an error and still exit 0.
func TestCommandsReturnErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "install invalid version", args: []string{"install", "not-a-version"}, wantErr: "invalid version"},
		{name: "install regex after version", args: []string{"install", "1.2.3", "x"}, wantErr: "a regex can only follow"},
		{name: "use invalid regex", args: []string{"use", "latest", "["}, wantErr: "invalid regex"},
		{name: "uninstall missing version", args: []string{"uninstall", "9.9.9"}, wantErr: "is not installed"},
		{name: "uninstall without a version", args: []string{"uninstall"}, wantErr: "specify the version to uninstall"},
		{name: "uninstall latest with nothing installed", args: []string{"uninstall", "latest"}, wantErr: "no local terraform versions found"},
		{name: "version-name without active version", args: []string{"version-name"}, wantErr: "failed to get current terraform version"},
		{name: "pin without active version", args: []string{"pin"}, wantErr: "failed to write current terraform version"},
		{name: "flavor invalid", args: []string{"flavor", "bogus"}, wantErr: "invalid flavor"},
		{name: "--flavor path traversal", args: []string{"--flavor", "../..", "list"}, wantErr: "invalid flavor"},
		{name: "list rejects extra args", args: []string{"list", "junk"}, wantErr: "unknown command"},
		{name: "pin rejects extra args", args: []string{"pin", "1.5.7"}, wantErr: "unknown command"},
		{name: "version-name rejects extra args", args: []string{"version-name", "x"}, wantErr: "unknown command"},
		{name: "list-remote rejects extra args", args: []string{"list-remote", "x"}, wantErr: "unknown command"},
		{name: "init rejects extra args", args: []string{"init", "x"}, wantErr: "unknown command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRoot(t)
			t.Chdir(t.TempDir())
			err := runCLI(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCommandsSucceed(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "list with nothing installed", args: []string{"list"}},
		{name: "flavor read", args: []string{"flavor"}},
		{name: "flavor write", args: []string{"flavor", "tofu"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRoot(t)
			t.Chdir(t.TempDir())
			if err := runCLI(t, tt.args...); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInstallUseVersionNameFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tfenvgo relies on symlinks and does not support Windows")
	}
	root := setupRoot(t)
	t.Chdir(t.TempDir())
	archive := makeZip(t, [2]string{"terraform", "#!/bin/sh\necho fake\n"})
	serveTerraformRelease(t, archive, sha256Hex(archive)+"  "+testArchiveName+"\n")

	if err := runCLI(t, "install", "1.5.7"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := runCLI(t, "use", "1.5.7"); err != nil {
		t.Fatalf("use: %v", err)
	}
	target, err := os.Readlink(filepath.Join(root, "bin", "terraform"))
	if err != nil {
		t.Fatalf("reading symlink: %v", err)
	}
	if target != installedBinaryPath(FlavorTerraform, "1.5.7") {
		t.Fatalf("symlink target = %q", target)
	}
	if err := runCLI(t, "version-name"); err != nil {
		t.Fatalf("version-name: %v", err)
	}
	if err := runCLI(t, "uninstall", "1.5.7"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
}

// TestFlavorCannotEscapeVersionsDir is the regression test for a flavor value
// such as "../../precious" making uninstall delete a directory outside
// ~/.tfenvgo.
func TestFlavorCannotEscapeVersionsDir(t *testing.T) {
	for _, viaEnv := range []bool{false, true} {
		root := setupRoot(t)
		t.Chdir(t.TempDir())
		precious := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-precious", "1.0.0")
		if err := os.MkdirAll(precious, 0o750); err != nil {
			t.Fatal(err)
		}
		escape := "../../" + filepath.Base(root) + "-precious"

		var err error
		if viaEnv {
			t.Setenv(flavorEnvKey, escape)
			err = runCLI(t, "uninstall", "1.0.0")
		} else {
			err = runCLI(t, "--flavor", escape, "uninstall", "1.0.0")
		}
		if err == nil || !strings.Contains(err.Error(), "invalid flavor") {
			t.Fatalf("viaEnv=%v: err = %v, want invalid flavor", viaEnv, err)
		}
		if _, statErr := os.Stat(precious); statErr != nil {
			t.Fatalf("viaEnv=%v: directory outside ~/.tfenvgo was touched: %v", viaEnv, statErr)
		}
	}
}

func TestListRemoteOutputIsPlain(t *testing.T) {
	setupRoot(t)
	t.Chdir(t.TempDir())
	mux := http.NewServeMux()
	mux.HandleFunc("/terraform", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<a href="/terraform/1.5.7/">x</a><a href="/terraform/1.10.0/">x</a>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	setReleaseURLs(t, srv.URL)

	var out bytes.Buffer
	if err := runCLIWithOutput(t, &out, "list-remote"); err != nil {
		t.Fatalf("list-remote: %v", err)
	}
	if got, want := out.String(), "1.10.0\n1.5.7\n"; got != want {
		t.Fatalf("output = %q, want %q (newest first, no prefixes or colors)", got, want)
	}
}

func TestUninstallActiveVersionRemovesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tfenvgo relies on symlinks and does not support Windows")
	}
	root := setupRoot(t)
	t.Chdir(t.TempDir())
	installFakeVersions(t, FlavorTerraform, "1.5.7", "1.6.0")
	if err := runCLI(t, "use", "1.5.7"); err != nil {
		t.Fatalf("use: %v", err)
	}
	link := filepath.Join(root, "bin", "terraform")

	// Uninstalling another version keeps the link.
	if err := runCLI(t, "uninstall", "1.6.0"); err != nil {
		t.Fatalf("uninstall 1.6.0: %v", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("symlink should remain after uninstalling an inactive version: %v", err)
	}

	// Uninstalling the active version removes it instead of leaving it dangling.
	if err := runCLI(t, "uninstall", "1.5.7"); err != nil {
		t.Fatalf("uninstall 1.5.7: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("symlink should be removed with the active version, lstat err = %v", err)
	}
}

func TestPinWritesActiveVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tfenvgo relies on symlinks and does not support Windows")
	}
	setupRoot(t)
	project := t.TempDir()
	t.Chdir(project)
	installFakeVersions(t, FlavorTerraform, "1.5.7")
	if err := runCLI(t, "use", "1.5.7"); err != nil {
		t.Fatalf("use: %v", err)
	}
	if err := runCLI(t, "pin"); err != nil {
		t.Fatalf("pin: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(project, terraformVersionFilename))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1.5.7\n" {
		t.Fatalf(".terraform-version = %q, want %q", got, "1.5.7\n")
	}
	if v, err := readVersionFromFile(); err != nil || v != "1.5.7" {
		t.Fatalf("readVersionFromFile() = %q, %v; want the pinned version back", v, err)
	}
}

func TestUseDoesNotLinkWhenInstallFails(t *testing.T) {
	root := setupRoot(t)
	t.Chdir(t.TempDir())
	archive := makeZip(t, [2]string{"terraform", "tampered"})
	serveTerraformRelease(t, archive, sha256Hex([]byte("other"))+"  "+testArchiveName+"\n")

	err := runCLI(t, "use", "1.5.7")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "bin", "terraform")); !os.IsNotExist(err) {
		t.Fatalf("symlink should not exist after a failed install, lstat err = %v", err)
	}
}

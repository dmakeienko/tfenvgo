package cmd

import (
	"io"
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
	rootCmd.SetArgs(args)
	rootCmd.SetOut(io.Discard)
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
		{name: "uninstall with nothing installed", args: []string{"uninstall"}, wantErr: "no local terraform versions found"},
		{name: "version-name without active version", args: []string{"version-name"}, wantErr: "failed to get current terraform version"},
		{name: "pin without active version", args: []string{"pin"}, wantErr: "failed to write current terraform version"},
		{name: "flavor invalid", args: []string{"flavor", "bogus"}, wantErr: "invalid flavor"},
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

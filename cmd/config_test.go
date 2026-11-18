package cmd

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type flavorCase struct {
	name       string
	flag       string
	env        string
	file       string
	want       string
	wantSource string
	wantErr    string
}

func TestFlavorResolution(t *testing.T) {
	tests := []flavorCase{
		{name: "built-in default", want: FlavorTerraform, wantSource: "built-in default"},
		{name: "file", file: "tofu\n", want: FlavorTofu, wantSource: "flavor"},
		{name: "env beats file", env: "terraform", file: "tofu", want: FlavorTerraform, wantSource: flavorEnvKey},
		{name: "flag beats env", flag: "tofu", env: "terraform", want: FlavorTofu, wantSource: "--flavor"},
		{name: "invalid file is ignored", file: "../..", want: FlavorTerraform, wantSource: "built-in default"},
		{name: "invalid flag", flag: "../..", wantErr: "--flavor: invalid flavor"},
		{name: "invalid env", env: "Terraform", wantErr: flavorEnvKey + ": invalid flavor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { checkFlavorCase(t, tt) })
	}
}

// checkFlavorCase sets up the flag/env/file inputs of tt and checks what
// flavorSource resolves.
func checkFlavorCase(t *testing.T, tt flavorCase) {
	t.Helper()
	root := setupRoot(t)
	if tt.env != "" {
		t.Setenv(flavorEnvKey, tt.env)
	}
	if tt.file != "" {
		writeTestFile(t, filepath.Join(root, "flavor"), tt.file)
	}

	got, source, err := flavorSource(tt.flag)
	if tt.wantErr != "" {
		assertErrContains(t, err, tt.wantErr)
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != tt.want || !strings.Contains(source, tt.wantSource) {
		t.Fatalf("got (%q, %q), want (%q, source containing %q)", got, source, tt.want, tt.wantSource)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %v, want containing %q", err, want)
	}
}

func TestEnsureOwnerExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	tests := []struct {
		name string
		from os.FileMode
		want os.FileMode
	}{
		{name: "adds only the owner execute bit", from: 0o640, want: 0o740},
		{name: "never widens group or other access", from: 0o600, want: 0o700},
		{name: "leaves an executable file alone", from: 0o750, want: 0o750},
	}
	for _, tt := range tests {
		path := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "_"))
		writeTestFile(t, path, "bin")
		if err := os.Chmod(path, tt.from); err != nil {
			t.Fatal(err)
		}
		if err := ensureOwnerExecutable(path); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != tt.want {
			t.Errorf("%s: mode %o -> %o, want %o", tt.name, tt.from, got, tt.want)
		}
	}
}

func TestInstallRejectsInvalidPlatform(t *testing.T) {
	for _, env := range []struct{ key, value string }{
		{archEnvKey, "amd64/../../evil"},
		{osTypeEnvKey, "Linux"},
		{archEnvKey, ""},
	} {
		setupRoot(t)
		t.Setenv(osTypeEnvKey, "linux")
		t.Setenv(archEnvKey, "amd64")
		t.Setenv(env.key, env.value)
		err := downloadBinary(FlavorTerraform, "1.5.7")
		if err == nil || !strings.Contains(err.Error(), "invalid "+env.key) {
			t.Fatalf("%s=%q: err = %v, want invalid %s", env.key, env.value, err, env.key)
		}
	}
}

func TestCheckRedirect(t *testing.T) {
	mustReq := func(raw string) *http.Request {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Request{URL: u}
	}
	httpsOrigin := []*http.Request{mustReq("https://releases.hashicorp.com/terraform")}

	if err := checkRedirect(mustReq("https://objects.githubusercontent.com/x"), httpsOrigin); err != nil {
		t.Fatalf("https -> https should be allowed: %v", err)
	}
	if err := checkRedirect(mustReq("http://evil.example/x"), httpsOrigin); err == nil {
		t.Fatal("https -> http should be refused")
	}
	if err := checkRedirect(mustReq("http://127.0.0.1/b"), []*http.Request{mustReq("http://127.0.0.1/a")}); err != nil {
		t.Fatalf("http -> http (local test servers) should be allowed: %v", err)
	}
	many := make([]*http.Request, maxRedirects)
	for i := range many {
		many[i] = mustReq("https://example.com/")
	}
	if err := checkRedirect(mustReq("https://example.com/next"), many); err == nil {
		t.Fatal("expected an error after too many redirects")
	}
}

func TestDownloadTimeout(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{"", defaultDownloadTimeout},
		{"600", 600 * time.Second},
		{"10m", 10 * time.Minute},
		{" 90s ", 90 * time.Second},
		{"0", defaultDownloadTimeout},
		{"-5", defaultDownloadTimeout},
		{"-1m", defaultDownloadTimeout},
		{"soon", defaultDownloadTimeout},
	}
	for _, tt := range tests {
		t.Setenv(downloadTimeoutEnvKey, tt.value)
		if got := downloadTimeout(); got != tt.want {
			t.Errorf("%s=%q: got %v, want %v", downloadTimeoutEnvKey, tt.value, got, tt.want)
		}
	}
}

func TestMigrateVersionsLayout(t *testing.T) {
	root := setupRoot(t)
	oldBin := filepath.Join(root, "versions", "1.5.7", "terraform")
	if err := os.MkdirAll(filepath.Dir(oldBin), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldBin, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A stray directory without a terraform binary must be left alone.
	if err := os.MkdirAll(filepath.Join(root, "versions", "notes"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := migrateVersionsLayout(); err != nil {
		t.Fatalf("migrateVersionsLayout: %v", err)
	}
	if _, err := os.Stat(installedBinaryPath(FlavorTerraform, "1.5.7")); err != nil {
		t.Fatalf("binary not migrated to the new layout: %v", err)
	}
	assertNotExist(t, filepath.Dir(oldBin))
	if _, err := os.Stat(filepath.Join(root, "versions", "notes")); err != nil {
		t.Fatalf("unrelated directory should not be touched: %v", err)
	}

	// Running it again is a no-op.
	if err := migrateVersionsLayout(); err != nil {
		t.Fatalf("second migrateVersionsLayout: %v", err)
	}
}

func TestReadVersionFromFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{name: "plain", content: "1.5.7\n", want: "1.5.7"},
		{name: "v prefix and whitespace", content: "  v1.5.7  \r\n", want: "v1.5.7"},
		{name: "first valid line wins", content: "# pinned\n1.6.0\n1.5.7\n", want: "1.6.0"},
		{name: "keyword is not a version", content: "latest\n", wantErr: true},
		{name: "empty", content: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, terraformVersionFilename), []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)
			got, err := readVersionFromFile()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %q, want an error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got (%q, %v), want %q", got, err, tt.want)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if _, err := readVersionFromFile(); err == nil {
			t.Fatal("expected an error without .terraform-version")
		}
	})
}

func TestVersionFromEnvOrFile(t *testing.T) {
	setupRoot(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, terraformVersionFilename), []byte("1.1.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	if got := versionFromEnvOrFile(); got != "1.1.1" {
		t.Fatalf("file only: got %q", got)
	}
	t.Setenv(terraformVersionEnvKey, "2.2.2")
	if got := versionFromEnvOrFile(); got != "2.2.2" {
		t.Fatalf("%s should beat the file: got %q", terraformVersionEnvKey, got)
	}
	t.Setenv(versionEnvKey, "3.3.3")
	if got := versionFromEnvOrFile(); got != "3.3.3" {
		t.Fatalf("%s should beat %s: got %q", versionEnvKey, terraformVersionEnvKey, got)
	}
}

func TestGetTerraformVersionConstraint(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.tf":      "resource \"null_resource\" \"x\" {}\n",
		"versions.tf":  "terraform {\n  required_version = \"~> 1.5.0\"\n}\n",
		"notes.txt":    "required_version = \"9.9.9\"\n",
		"override.tfx": "required_version = \"8.8.8\"\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	got, err := getTerraformVersionConstraint()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "~> 1.5.0" {
		t.Fatalf("got %q, want %q (only *.tf files count)", got, "~> 1.5.0")
	}
}

func TestColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if colorEnabled() {
		t.Fatal("colors should be disabled when NO_COLOR is set")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if colorEnabled() {
		t.Fatal("colors should be disabled for TERM=dumb")
	}
}

func TestBinDirOnPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bin")
	t.Setenv("PATH", strings.Join([]string{"/usr/bin", dir + string(filepath.Separator)}, string(filepath.ListSeparator)))
	if !binDirOnPath(dir) {
		t.Fatal("expected dir (with trailing separator in PATH) to be found")
	}
	t.Setenv("PATH", "/usr/bin")
	if binDirOnPath(dir) {
		t.Fatal("dir is not on PATH")
	}
}

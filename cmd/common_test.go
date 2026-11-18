package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseVersionArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		fallback      string
		wantRequested string
		wantPattern   string
		wantErr       string
	}{
		{name: "no args, empty fallback", args: nil, fallback: "", wantRequested: latestArg},
		{name: "no args, fallback used", args: nil, fallback: "1.5.7", wantRequested: "1.5.7"},
		{name: "explicit version", args: []string{"1.2.3"}, fallback: "9.9.9", wantRequested: "1.2.3"},
		{name: "latest with regex", args: []string{"latest", `^1\.5`}, wantRequested: latestArg, wantPattern: `^1\.5`},
		{name: "regex after non-latest", args: []string{"1.2.3", "x"}, wantErr: "a regex can only follow"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requested, pattern, err := parseVersionArgs(tt.args, func() string { return tt.fallback })
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if requested != tt.wantRequested || pattern != tt.wantPattern {
				t.Fatalf("got (%q, %q), want (%q, %q)", requested, pattern, tt.wantRequested, tt.wantPattern)
			}
		})
	}
}

func TestResolveVersionLocal(t *testing.T) {
	setupRoot(t)
	installFakeVersions(t, FlavorTerraform, "1.4.0", "1.5.7", "1.5.10", "1.6.0-rc1", "1.10.0")

	allKeywords := []string{latestArg, latestAllowedArg, minRequiredArg}
	tests := []struct {
		name       string
		requested  string
		pattern    string
		keywords   []string
		prerelease bool
		want       string
		wantErr    string
	}{
		// "1.5.10" < "1.10.0" only under semver ordering, not string ordering.
		{name: "latest is semver-sorted", requested: latestArg, want: "1.10.0"},
		{name: "regex selects newest match", requested: latestArg, pattern: `^1\.5\.`, want: "1.5.10"},
		{name: "regex with alternation", requested: latestArg, pattern: `^1\.(4|6)\.`, want: "1.4.0"},
		{name: "regex sees prereleases when asked", requested: latestArg, pattern: `^1\.6\.`, prerelease: true, want: "1.6.0-rc1"},
		{name: "regex that is not a semver constraint", requested: latestArg, pattern: `^1\.5\..*$`, want: "1.5.10"},
		{name: "exact version passes through", requested: "1.2.3", want: "1.2.3"},
		{name: "regex without matches", requested: latestArg, pattern: `^2\.`, wantErr: "no local terraform version matches"},
		{name: "invalid regex errors instead of panicking", requested: latestArg, pattern: `[`, wantErr: "invalid regex"},
		{name: "not a version", requested: "not-a-version", wantErr: "invalid version"},
		{name: "keyword not allowed for command", requested: latestAllowedArg, keywords: []string{latestArg}, wantErr: "invalid version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := tt.keywords
			if keywords == nil {
				keywords = allKeywords
			}
			got, err := resolveVersion(FlavorTerraform, tt.requested, tt.pattern, targetLocal, keywords, tt.prerelease)
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

func TestConstraintKeywordsLocal(t *testing.T) {
	setupRoot(t)
	installFakeVersions(t, FlavorTerraform, "1.4.0", "1.5.7", "1.6.0", "1.10.0")

	project := t.TempDir()
	tf := "terraform {\n  required_version = \">= 1.5.0, < 1.10.0\"\n}\n"
	if err := os.WriteFile(filepath.Join(project, "main.tf"), []byte(tf), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	keywords := []string{latestArg, latestAllowedArg, minRequiredArg}
	for requested, want := range map[string]string{
		latestAllowedArg: "1.6.0",
		minRequiredArg:   "1.5.7",
	} {
		got, err := resolveVersion(FlavorTerraform, requested, "", targetLocal, keywords, false)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", requested, err)
		}
		if got != want {
			t.Fatalf("%s: got %q, want %q", requested, got, want)
		}
	}
}

func TestConstraintKeywordsWithoutRequiredVersion(t *testing.T) {
	setupRoot(t)
	installFakeVersions(t, FlavorTerraform, "1.5.7")
	t.Chdir(t.TempDir())

	keywords := []string{latestArg, latestAllowedArg, minRequiredArg}
	for _, requested := range []string{latestAllowedArg, minRequiredArg} {
		_, err := resolveVersion(FlavorTerraform, requested, "", targetLocal, keywords, false)
		if err == nil || !strings.Contains(err.Error(), "required_version not found") {
			t.Fatalf("%s: err = %v, want required_version not found", requested, err)
		}
	}
}

func TestGetLocalVersionsWithNothingInstalled(t *testing.T) {
	setupRoot(t)
	versions, err := getLocalVersions(FlavorTerraform, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 0 {
		t.Fatalf("got %v, want no versions", versions)
	}
}

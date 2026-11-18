package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoteTerraformVersionsAreSemverSorted(t *testing.T) {
	setupRoot(t)
	// Deliberately not in descending order.
	listing := `<html><body><ul>
<li><a href="/terraform/1.5.7/">terraform_1.5.7</a></li>
<li><a href="/terraform/1.10.0/">terraform_1.10.0</a></li>
<li><a href="/terraform/1.11.0-rc1/">terraform_1.11.0-rc1</a></li>
<li><a href="/terraform/1.9.8/">terraform_1.9.8</a></li>
<li><a href="https://example.com/">unrelated</a></li>
</ul></body></html>`
	mux := http.NewServeMux()
	mux.HandleFunc("/terraform", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, listing)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	setReleaseURLs(t, srv.URL)

	keywords := []string{latestArg}
	tests := []struct {
		name       string
		pattern    string
		prerelease bool
		want       string
	}{
		{name: "latest stable", want: "1.10.0"},
		{name: "latest including prereleases", prerelease: true, want: "1.11.0-rc1"},
		{name: "latest matching regex", pattern: `^1\.9\.`, want: "1.9.8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveVersion(FlavorTerraform, latestArg, tt.pattern, targetRemote, keywords, tt.prerelease)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoteTofuVersionsPaginate(t *testing.T) {
	setupRoot(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/tofu/releases", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = io.WriteString(w, `[{"tag_name":"v1.8.0","prerelease":false},{"tag_name":"v1.11.0-beta1","prerelease":true}]`)
		case "2":
			_, _ = io.WriteString(w, `[{"tag_name":"v1.10.1","prerelease":false},{"tag_name":"not-a-version","prerelease":false}]`)
		default:
			_, _ = io.WriteString(w, `[]`)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	setReleaseURLs(t, srv.URL)

	got, err := resolveVersion(FlavorTofu, latestArg, "", targetRemote, []string{latestArg}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.10.1" {
		t.Fatalf("got %q, want 1.10.1 (from page 2)", got)
	}

	got, err = resolveVersion(FlavorTofu, latestArg, "", targetRemote, []string{latestArg}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.11.0-beta1" {
		t.Fatalf("got %q, want 1.11.0-beta1 with prereleases", got)
	}
}

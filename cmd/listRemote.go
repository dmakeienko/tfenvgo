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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

// getRemoteVersions dispatches to the flavor-specific implementation.
func getRemoteVersions(flv string, includePrerelease bool) ([]string, error) {
	switch flv {
	case FlavorTofu:
		return getRemoteTofuVersions(includePrerelease)
	default:
		return getRemoteTerraformVersions(includePrerelease)
	}
}

func getRemoteTerraformVersions(preReleaseVersionsIncluded bool) ([]string, error) {
	// Create HTTP client with security configurations
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", terraformReleasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", "tfenvgo/"+Version)

	// #nosec G704 - URL is hardcoded terraformReleasesURL
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page, status code: %d", resp.StatusCode)
	}

	var versions []string
	var versionRegex *regexp.Regexp

	stableVersionRegex := regexp.MustCompile(`^/terraform/([0-9]+\.[0-9]+\.[0-9]+)/$`)
	preReleaseVersionRegex := regexp.MustCompile(`^/terraform/(\d+\.\d+\.\d+(-[a-z]+\d+)?)\/$`)

	if preReleaseVersionsIncluded {
		versionRegex = preReleaseVersionRegex
	} else {
		versionRegex = stableVersionRegex
	}

	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken {
			tagName, hasAttr := z.TagName()
			if string(tagName) == "a" && hasAttr {
				for {
					attrName, attrValue, moreAttr := z.TagAttr()
					if string(attrName) == "href" {
						matches := versionRegex.FindStringSubmatch(string(attrValue))
						if len(matches) >= 2 {
							versions = append(versions, matches[1])
						}
					}
					if !moreAttr {
						break
					}
				}
			}
		}
	}

	return versions, nil
}

// getRemoteTofuVersions fetches OpenTofu releases from the GitHub API with pagination support.
func getRemoteTofuVersions(includePrerelease bool) ([]string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var allReleases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
	}

	page := 1
	for {
		url := fmt.Sprintf("%s?per_page=100&page=%d", tofuReleasesAPI, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "tfenvgo/"+Version)
		req.Header.Set("Accept", "application/vnd.github+json")

		// #nosec G704 - URL is constructed from hardcoded tofuReleasesAPI constant
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tofu releases: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch tofu releases, status: %d", resp.StatusCode)
		}

		var pageReleases []struct {
			TagName    string `json:"tag_name"`
			Prerelease bool   `json:"prerelease"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&pageReleases); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode tofu releases: %w", err)
		}
		resp.Body.Close()

		if len(pageReleases) == 0 {
			break
		}

		allReleases = append(allReleases, pageReleases...)
		page++
	}

	if len(allReleases) == 0 {
		return nil, fmt.Errorf("no releases found from OpenTofu GitHub API")
	}

	var versions []*semver.Version
	for _, r := range allReleases {
		if r.Prerelease && !includePrerelease {
			continue
		}
		tag := strings.TrimPrefix(r.TagName, "v")
		v, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		// Skip pre-release semver versions unless requested
		if !includePrerelease && v.Prerelease() != "" {
			continue
		}
		versions = append(versions, v)
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))

	var result []string
	for _, v := range versions {
		result = append(result, v.String())
	}
	return result, nil
}

// listRemoteCmd represents the listRemote command
var listRemoteCmd = &cobra.Command{
	Use:   "list-remote",
	Short: "List all available versions from the remote registry",
	Long:  "List all available versions from the remote registry (Terraform or OpenTofu depending on --flavor)",
	Run: func(cmd *cobra.Command, args []string) {
		flv := resolveFlavor(flavorFlag)
		versions, err := getRemoteVersions(flv, PreReleaseVersionsIncluded)
		if err != nil {
			LogError("Failed to get versions: %v", err)
			return
		}
		LogInfo("Available %s versions:", flv)
		for _, v := range versions {
			LogInfo("%s", v)
		}
	},
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
	listRemoteCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

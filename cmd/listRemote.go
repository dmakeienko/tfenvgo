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
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

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

// listRemoteCmd represents the listRemote command
var listRemoteCmd = &cobra.Command{
	Use:   "list-remote",
	Short: "List all available Terraform versions",
	Long:  "List all available Terraform versions",
	Run: func(cmd *cobra.Command, args []string) {
		versions, err := getRemoteTerraformVersions(PreReleaseVersionsIncluded)
		if err != nil {
			LogError("Failed to get versions: %v", err)
			return
		}
		LogInfo("Available versions:")
		for _, v := range versions {
			LogInfo("%s", v)
		}
	},
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
	listRemoteCmd.Flags().BoolVarP(&PreReleaseVersionsIncluded, "include-prerelease", "", false, "Include pre-release versions")
}

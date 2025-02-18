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
	"fmt"
	"net/http"
	"regexp"

	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

func getRemoteTerraformVersions() ([]string, error) {
	resp, err := http.Get(terraformReleasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page, status code: %d", resp.StatusCode)
	}

	var versions []string
	z := html.NewTokenizer(resp.Body)
	versionRegex := regexp.MustCompile(`^/terraform/([0-9]+\.[0-9]+\.[0-9]+)/$`)

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
						if len(matches) == 2 {
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
	Short: "List all available (stable) Terraform versions",
	Long:  "List all available (stable) Terraform versions",
	Run: func(cmd *cobra.Command, args []string) {
		versions, err := getRemoteTerraformVersions()
		if err != nil {
			fmt.Println("Failed to get versions:", err)
			return
		}
		fmt.Println(Green + "Available stable versions:" + Reset)
		for _, v := range versions {
			fmt.Println(Gray + v + Reset)
		}
	},
}

func init() {
	rootCmd.AddCommand(listRemoteCmd)
}

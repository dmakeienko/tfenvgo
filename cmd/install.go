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
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func unarchiveZip(archivePath, version string) {
	dst := terraformVersionPath + "/" + version
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory..." + filePath)
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				fmt.Println("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func downloadTerraform(version string) error {
	terraformDownloadURL := terraformReleasesURL + "/" + version + "/terraform_" + version + "_" + osType + "_" + arch + ".zip"
	fmt.Println("Downloading " + terraformDownloadURL)

	// Get the data
	resp, err := http.Get(terraformDownloadURL) //nolint
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	filepath := "/tmp/" + path.Base(resp.Request.URL.String())

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	fmt.Println("Downloaded file to " + filepath)

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	unarchiveZip(filepath, version)
	println("Removing" + filepath)
	os.Remove(filepath)
	return err
}

func installTerraform(version string) {
	_, err := os.Stat(terraformVersionPath + "/" + version)
	if os.IsNotExist(err) {
		err := downloadTerraform(version)
		if err != nil {
			fmt.Println("failed to download binary: %w", err)
		}
	} else {
		fmt.Println(Yellow + "Version " + version + " is already installed." + Reset)
	}
	// useVersion(version)  // Do I need to change version after download or use expilicitly?
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Installs a specific Terraform version",
	Run: func(cmd *cobra.Command, args []string) {
		installTerraform(args[0])
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

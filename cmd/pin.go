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
	"os"

	"github.com/spf13/cobra"
)

func writeCurrentVersionToFile() error {
	currentVersion, err := getCurrentTerraformVersion()
	if err != nil {
		return fmt.Errorf("failed to get current terraform version: %w", err)
	}

	terraformVersionFile, err := os.Create(terraformVersionFilename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer terraformVersionFile.Close()

	_, err2 := terraformVersionFile.WriteString(currentVersion)
	if err2 != nil {
		return fmt.Errorf("failed to write  terraform version to file: %w", err2)
	}
	fmt.Println(Green + terraformVersionFilename + " file created with current terraform version: " + currentVersion + Reset)
	return nil
}

// pinCmd represents the pin command
var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: "Write the current active version to .terraform-version file",
	Run: func(cmd *cobra.Command, args []string) {
		err := writeCurrentVersionToFile()
		if err != nil {
			fmt.Println(Red + "Failed to write current terraform version to file: " + err.Error() + Reset)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(pinCmd)
}

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
	"path/filepath"

	"github.com/spf13/cobra"
)

func initConfig() error {
	if _, err := os.Stat(terraformBinPath); os.IsNotExist(err) {
		err := os.MkdirAll(terraformBinPath, 0o750)
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		LogInfo("%s has been created successfully.", terraformBinPath)
	}
	return nil
}

// binDirOnPath reports whether dir is one of the entries in $PATH.
func binDirOnPath(dir string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(entry) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create required configuration for tfenvgo to work",
	Long: `Create ~/.tfenvgo/bin, where "tfenvgo use" puts the active terraform and tofu
symlinks, and check that it is on your PATH. Running it is optional: the other
commands create the directory when they need it.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initConfig(); err != nil {
			return err
		}
		if !binDirOnPath(terraformBinPath) {
			LogWarn("%s is not on your PATH. Add this line to your shell profile:", terraformBinPath)
			fmt.Println(`  export PATH="$HOME/.tfenvgo/bin:$PATH"`)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

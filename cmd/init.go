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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func initConfig() error {
	binPath := os.Getenv("HOME") + "/.tfenvgo/bin"

	err := os.MkdirAll(binPath, os.ModePerm)

	if err != nil {
		return errors.New("failed to create  config")
	}
	fmt.Println(binPath + " has been created successfully.")

	terraformBinPath := "export" + " PATH=$PATH:" + binPath
	shellConfigPath := os.Getenv("HOME") + "/.zshrc"

	file, err := os.Open(shellConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Check if the line already exists in the file.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == terraformBinPath {
			fmt.Println(Gray + "Shell config contains required configuration. No changes to will be made." + Reset)
			fmt.Println(Gray + "If you encounter any problems, try to delete line \"" + Yellow + terraformBinPath + Gray + "\" from " + Yellow + shellConfigPath + Gray + " and run \"" + Yellow + "tfenvgo init" + Yellow + "\" again." + Reset)
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// If the line is not found, append it to the file.
	file, err = os.OpenFile(shellConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shell config for appending: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(terraformBinPath + "\n"); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Println("Shell configuration has been updated. Please restart your shell to apply changes.")
	return err
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates required configuration for tfenvgo to work",
	Long:  "Creates required folders and files and update shell configuration to be able to use tfenvgo.",
	Run: func(cmd *cobra.Command, args []string) {
		err := initConfig()
		if err != nil {
			fmt.Println("failed to create config: %w", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

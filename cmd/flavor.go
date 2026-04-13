/*
Copyright © 2026 Denys Makeienko <denys.makeienko@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// flavorCmd represents the flavor command.
// With no args it prints the current resolved flavor and its source.
// With one arg (tofu|terraform) it writes that flavor as the persistent default.
var flavorCmd = &cobra.Command{
	Use:   "flavor [tofu|terraform]",
	Short: "Get or set the default binary flavor (tofu or terraform)",
	Long: `Get or set the default binary flavor.

Without arguments, prints the currently resolved flavor and where it comes from
(--flavor flag, TFENVGO_FLAVOR env var, ~/.tfenvgo/flavor file, or built-in default).

With one argument, writes the chosen flavor to ~/.tfenvgo/flavor so it becomes
the persistent default for all future invocations.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			LogError("Failed to initialise config: %v", err)
			return
		}

		if len(args) == 0 {
			// Read mode: report the resolved flavor and its source.
			printResolvedFlavor()
			return
		}

		// Write mode: validate and persist.
		requested := args[0]
		if requested != FlavorTerraform && requested != FlavorTofu {
			LogError("Invalid flavor %q. Allowed values: %s, %s", requested, FlavorTerraform, FlavorTofu)
			return
		}

		flavorFilePath := filepath.Join(rootURL, "flavor")
		if err := os.WriteFile(flavorFilePath, []byte(requested), 0o600); err != nil {
			LogError("Error writing flavor file: %v", err)
			return
		}
		fmt.Printf("Default flavor set to '%s'\n", requested)
	},
}

// printResolvedFlavor prints the effective flavor and explains where it came from.
func printResolvedFlavor() {
	if flavorFlag != "" {
		fmt.Printf("Flavor: " + Green + "%s " + Gray + "(source: --flavor flag)\n", flavorFlag)
		return
	}
	if env := os.Getenv(flavorEnvKey); env != "" {
		fmt.Printf("Flavor: %s (source: %s env var)\n", env, flavorEnvKey)
		return
	}
	flavorFile := filepath.Join(rootURL, "flavor")
	data, err := os.ReadFile(flavorFile) // #nosec G304 - path constructed from controlled rootURL
	if err == nil {
		f := strings.TrimSpace(string(data))
		if f == FlavorTofu || f == FlavorTerraform {
			fmt.Printf("Flavor: " + Green + "%s " + Gray + "(source: %s)\n", f, flavorFile)
			return
		}
	}
	fmt.Printf("Flavor: " + Green +  "%s " + Gray + "(source: built-in default)\n", FlavorTerraform)
}

func init() {
	rootCmd.AddCommand(flavorCmd)

	// Persistent flag available on every subcommand.
	rootCmd.PersistentFlags().StringVar(&flavorFlag, "flavor", "", "binary flavor: terraform or tofu (overrides default)")
}

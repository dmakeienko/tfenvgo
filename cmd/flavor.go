/*
Copyright © 2026 Denys Makeienko <denys.makeienko@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	Example: `  tfenvgo flavor
  tfenvgo flavor tofu
  TFENVGO_FLAVOR=tofu tfenvgo list`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{FlavorTerraform, FlavorTofu},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Read mode: report the resolved flavor and its source.
			flv, source, err := flavorSource(flavorFlag)
			if err != nil {
				return err
			}
			fmt.Printf("Flavor: "+Green+"%s "+Gray+"(source: %s)"+Reset+"\n", flv, source)
			return nil
		}

		// Write mode: validate and persist.
		requested := args[0]
		if err := validateFlavor(requested); err != nil {
			return err
		}
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialise config: %w", err)
		}

		flavorFilePath := filepath.Join(rootURL, "flavor")
		if err := os.WriteFile(flavorFilePath, []byte(requested), 0o600); err != nil {
			return fmt.Errorf("error writing flavor file: %w", err)
		}
		fmt.Printf("Default flavor set to "+Green+"'%s'\n"+Reset, requested)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(flavorCmd)

	// Persistent flag available on every subcommand.
	rootCmd.PersistentFlags().StringVar(&flavorFlag, "flavor", "", "binary flavor: terraform or tofu (overrides default)")
}

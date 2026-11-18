package cmd

import "github.com/spf13/cobra"

// versionArgHelp documents the VERSION argument shared by install and use.
const versionArgHelp = `VERSION can be one of:
  x.y.z            an exact version, e.g. 1.5.7 or 1.12.0-alpha20250213
  latest           the newest stable release
  latest REGEX     the newest release whose version matches the Go regular
                   expression REGEX, e.g. latest '^1\.5\.' (quote it for the shell)
  latest-allowed   the newest release allowed by required_version in ./*.tf
  min-required     the oldest release allowed by required_version in ./*.tf

Add --include-prerelease to let latest and latest REGEX pick pre-releases.

Without VERSION, the first of these that is set is used: TFENVGO_VERSION,
TFENVGO_TERRAFORM_VERSION, the version in ./.terraform-version, otherwise latest.`

// completeVersions returns a shell-completion function suggesting keywords
// and, when installed is true, the installed versions of the current flavor.
// It never hits the network, so completion stays fast and offline.
func completeVersions(keywords []string, installed bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		suggestions := append([]string{}, keywords...)
		if installed {
			if flv, err := resolveFlavor(flavorFlag); err == nil {
				if versions, err := getLocalVersions(flv, true); err == nil {
					suggestions = append(suggestions, versions...)
				}
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	}
}

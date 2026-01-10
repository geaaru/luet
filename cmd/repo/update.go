/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_repo

import (
	"os"

	cfg "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"
	wagon "github.com/macaroni-os/anise/pkg/v2/repository"

	"github.com/spf13/cobra"
)

func NewRepoUpdateCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "update [repo1] [repo2] [OPTIONS]",
		Short: "Update a specific cached repository or all cached repositories.",
		Example: `
# Update all cached repositories:
$> anise repo update

# Update only repo1 and repo2
$> anise repo update repo1 repo2
`,
		Aliases: []string{"up"},
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		Run: func(cmd *cobra.Command, args []string) {
			ignore, _ := cmd.Flags().GetBool("ignore-errors")
			force, _ := cmd.Flags().GetBool("force")

			opts := &wagon.SyncOpts{
				Force:        force,
				IgnoreErrors: ignore,
			}

			rails := wagon.NewWagonsRails(config)
			err := rails.SyncRepos(args, opts)
			if err != nil {
				Error(err.Error())
				os.Exit(1)
			}
		},
	}

	ans.Flags().BoolP("ignore-errors", "i", false,
		"Ignore errors on sync repositories.")
	ans.Flags().BoolP("force", "f", false, "Force resync.")

	return ans
}

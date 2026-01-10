/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/macaroni-os/anise/cmd/database"
	config "github.com/macaroni-os/anise/pkg/config"

	"github.com/spf13/cobra"
)

func newDatabaseCommand(cfg *config.AniseConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "database [command] [OPTIONS]",
		Short: "Manage system database (dangerous commands ahead!)",
		Long: `Allows to manipulate Anise internal database of installed packages. Use with caution!

	Removing packages by hand from the database can result in a broken system, and thus it's not reccomended.
	`,
	}

	ans.AddCommand(
		NewDatabaseCreateCommand(cfg),
		NewDatabaseGetCommand(cfg),
		NewDatabaseRemoveCommand(cfg),
		NewDatabaseReindexCommand(cfg),
	)

	return ans
}

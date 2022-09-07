/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_database

import (
	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func NewDatabaseReindexCommand(cfg *config.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "reindex",
		Short: "Reindex local database",
		Args:  cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {

			aManager := installer.NewArtifactsManager(cfg)
			defer aManager.Close()

			aManager.Setup()

			err := aManager.Database.RebuildIndexes()
			if err != nil {
				Fatal("Error on recreate indexes: " + err.Error())
			}
		},
	}

	return ans
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_database

import (
	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func NewDatabaseRemoveCommand(cfg *config.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "remove [package1] [package2] ...",
		Short: "Remove a package from the system DB (forcefully - you normally don't want to do that)",
		Long: `Removes a package in the system database without actually uninstalling it:

		$ luet database remove foo/bar

This commands takes multiple packages as arguments and prunes their entries from the system database.
`,
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {

			aManager := installer.NewArtifactsManager(cfg)
			defer aManager.Close()

			aManager.Setup()

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}

				if err := aManager.Database.RemovePackage(pack); err != nil {
					Fatal("Failed removing ", a, ": ", err.Error())
				}

				if err := aManager.Database.RemovePackageFiles(pack); err != nil {
					Fatal("Failed removing files for ", a, ": ", err.Error())
				}
				if err := aManager.Database.RemovePackageFinalizer(pack); err != nil {
					Warning("Failed removing finalizer for ", a, ": ", err.Error())
				}

			}

		},
	}

	return ans
}

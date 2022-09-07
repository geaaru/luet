/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_database

import (
	"fmt"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewDatabaseGetCommand(cfg *config.LuetConfig) *cobra.Command {
	var c = &cobra.Command{
		Use:   "get <package>",
		Short: "Get a package in the system DB as yaml",
		Long: `Get a package in the system database in the YAML format:

		$ luet database get system/foo

To return also files:
		$ luet database get --files system/foo`,
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			showFiles, _ := cmd.Flags().GetBool("files")

			aManager := installer.NewArtifactsManager(cfg)
			defer aManager.Close()

			aManager.Setup()

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					continue
				}

				ps, err := aManager.Database.FindPackages(pack)
				if err != nil {
					continue
				}
				for _, p := range ps {
					y, err := p.Yaml()
					if err != nil {
						continue
					}
					fmt.Println(string(y))
					if showFiles {
						files, err := aManager.Database.GetPackageFiles(p)
						if err != nil {
							continue
						}
						b, err := yaml.Marshal(files)
						if err != nil {
							continue
						}
						fmt.Println("files:\n" + string(b))
					}
				}
			}
		},
	}
	c.Flags().Bool("files", false, "Show package files.")

	return c
}

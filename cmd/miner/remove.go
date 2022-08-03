/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package miner

import (
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	util "github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	installer "github.com/geaaru/luet/pkg/v2/installer"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/spf13/cobra"
)

func NewRemovePackage(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "remove-package <pkg1> <pkg2> ... <pkgN>",
		Short:   `Uninstall a package without analyze deps and conflicts and in the passed order.`,
		Aliases: []string{"r", "rm"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Missing arguments.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			pkgs := []*pkg.DefaultPackage{}
			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")

			for _, pstr := range args {
				p, err := helpers.ParsePackageStr(pstr)
				if err != nil {
					fmt.Println("Error on parse package string " + pstr + ": " +
						err.Error())
					os.Exit(1)
				}

				pkgs = append(pkgs, p)
			}

			searchOpts := &wagon.StonesSearchOpts{
				Packages:      pkgs,
				Categories:    []string{},
				Labels:        []string{},
				LabelsMatches: []string{},
				Matches:       []string{},
				FilesOwner:    []string{},
				Hidden:        true,
				AndCondition:  false,
				WithFiles:     true,
			}

			stones, err := util.SearchInstalled(config, searchOpts)
			if err != nil {
				Error(err.Error())
				os.Exit(1)
			}

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			fail := false

			for _, s := range *stones {
				err = aManager.RemovePackage(
					s, config.GetSystem().Rootfs,
					preserveSystem,
				)
				if err != nil {
					fail = true
					fmt.Println(fmt.Sprintf(
						"Error on install artifact %s: %s",
						s.HumanReadableString(),
						err.Error()))
					Error(fmt.Sprintf("[%40s] uninstall failed - :fire:", s.HumanReadableString()))
				}

			}

			if len(*stones) == 0 {
				Warning("No packages found.")
				fail = true
			} else if len(*stones) != len(pkgs) {
				Warning("Not all packages found.")
				fail = true
			}

			if fail {
				os.Exit(1)
			}
		},
	}

	flags := ans.Flags()

	flags.String("system-dbpath", "", "System db path")
	flags.String("system-target", "", "System rootpath")
	flags.String("system-engine", "", "System DB engine")
	flags.Bool("preserve-system-essentials", true, "Preserve system essentials files.")

	return ans
}

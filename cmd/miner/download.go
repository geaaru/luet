/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package miner

import (
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
	installer "github.com/geaaru/luet/pkg/v2/installer"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/spf13/cobra"
)

func NewDownload(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "download <repository-name> <pkg1> <pkg2> ... <pkgN>",
		Short:   "Download a package from a specified repository.",
		Aliases: []string{"d"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("Missing arguments.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {

			rname := args[0]
			pkgs := []*pkg.DefaultPackage{}

			repo, err := config.GetSystemRepository(rname)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			args = args[1:]
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

			repobasedir := config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
			r := wagon.NewWagonRepository(repo)
			err = r.ReadWagonIdentify(repobasedir)
			if err != nil {
				Fatal("Error on read repository identity file: " + err.Error())
			}
			artifactsRef, err := r.SearchArtifacts(searchOpts)
			if err != nil {
				Warning("Error on read repository catalog for repo : " + r.Identity.Name)
				os.Exit(1)
			}

			r.ClearCatalog()

			aManager := installer.NewArtifactsManager(config)

			fail := false

			artifacts := *artifactsRef
			for _, a := range artifacts {
				err = aManager.DownloadPackage(a, r)
				if err != nil {
					fail = true
					fmt.Println(fmt.Sprintf(
						"Error on download artifact %s: %s",
						a.Runtime.HumanReadableString(),
						err.Error()))
					Error(fmt.Sprintf("[%40s] :fire:", a.Runtime.HumanReadableString()))
				} else {
					Info(fmt.Sprintf("[%40s] :check_mark:", a.Runtime.HumanReadableString()))
				}
			}

			if len(artifacts) == 0 {
				Warning("No packages found.")
				fail = true
			} else if len(artifacts) != len(args) {
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
	flags.String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	flags.Float32("solver-rate", 0.7, "Solver learning rate")
	flags.Float32("solver-discount", 1.0, "Solver discount rate")
	flags.Int("solver-attempts", 9000, "Solver maximum attempts")

	return ans
}

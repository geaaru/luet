/*
Copyright © 2022-2023 Macaroni OS Linux
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
	installer "github.com/geaaru/luet/pkg/v2/installer"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	"github.com/geaaru/luet/pkg/v2/repository/mask"
	"github.com/logrusorgru/aurora"

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
				p, err := helpers.ParsePackageStr(config, pstr)
				if err != nil {
					fmt.Println("Error on parse package string " + pstr + ": " +
						err.Error())
					os.Exit(1)
				}

				pkgs = append(pkgs, p)
			}

			// On miner I don't load masks. Keep full control to users.
			maskManager := mask.NewPackagesMaskManager(config)

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
				IgnoreMasks:   false,
			}

			repobasedir := config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
			r := wagon.NewWagonRepository(repo)
			err = r.ReadWagonIdentify(repobasedir)
			if err != nil {
				Fatal("Error on read repository identity file: " + err.Error())
			}
			artifactsRef, err := r.SearchArtifacts(searchOpts, maskManager)
			if err != nil {
				Warning("Error on read repository catalog for repo : " + r.Identity.Name)
				os.Exit(1)
			}

			r.ClearCatalog()

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			fail := false

			artifacts := *artifactsRef
			ndownloads := len(artifacts)
			for idx, a := range artifacts {

				msg := fmt.Sprintf(
					"[%3d of %3d] %-65s - %-15s",
					aurora.Bold(aurora.BrightMagenta(idx+1)),
					aurora.Bold(aurora.BrightMagenta(ndownloads)),
					fmt.Sprintf("%s::%s", a.GetPackage().PackageName(),
						a.GetPackage().Repository,
					),
					a.GetPackage().GetVersion())

				err = aManager.DownloadPackage(a, r, msg)
				if err != nil {
					fail = true
					fmt.Println(fmt.Sprintf(
						"Error on download artifact %s: %s",
						a.Runtime.HumanReadableString(),
						err.Error()))
					Error(fmt.Sprintf(":package:%s # download failed :fire:", msg))
				} else {
					Info(fmt.Sprintf(":package:%s # downloaded :check_mark:", msg))
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

	return ans
}

/*
Copyright © 2022-2023 Macaroni OS Linux
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
	"github.com/geaaru/luet/pkg/subsets"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	installer "github.com/geaaru/luet/pkg/v2/installer"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	"github.com/logrusorgru/aurora"

	"github.com/spf13/cobra"
)

// TODO: Replace this when repository is inside the package.
type StonePolished struct {
	Stone    *wagon.Stone
	Artifact *artifact.PackageArtifact
}

func NewReinstallPackage(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "reinstall-package <pkg1> <pkg2> ... <pkgN>",
		Short:   `Reinstall a package without analyze deps and conflicts and in the passed order. The package must be available with the same version of a repository.`,
		Aliases: []string{"ri"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Missing arguments.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			pkgs := []*pkg.DefaultPackage{}

			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			force, _ := cmd.Flags().GetBool("force")

			for _, pstr := range args {
				p, err := helpers.ParsePackageStr(config, pstr)
				if err != nil {
					fmt.Println("Error on parse package string " + pstr + ": " +
						err.Error())
					os.Exit(1)
				}

				pkgs = append(pkgs, p)
			}

			// Load finalizer runtime environments
			err := util.SetCliFinalizerEnvs(finalizerEnvs)
			if err != nil {
				Fatal(err.Error())
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
				// Needed for the uninstall
				WithFiles:        true,
				WithRootfsPrefix: false,
			}

			searcher := wagon.NewSearcherSimple(config)
			stones, err := searcher.SearchInstalled(searchOpts)
			searcher.Close()
			if err != nil {
				Error(err.Error())
				os.Exit(1)
			}

			// Searching the packages over the existing repos.
			reposArtifacts, err := searcher.SearchArtifacts(searchOpts)
			if err != nil {
				Error(err.Error())
				os.Exit(1)
			}

			// Convert list to map
			aPack := &artifact.ArtifactsPack{
				Artifacts: (*reposArtifacts),
			}
			mapArtifacts := aPack.ToMap()
			aPack = nil
			reposArtifacts = nil

			// Load config protect configs
			installer.LoadConfigProtectConfs(config)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			fail := false
			errQueue := []error{}

			mapRepos := make(map[string]*wagon.WagonRepository, 0)
			pkgsQueue := []*StonePolished{}

			for _, s := range *stones {
				pMatch := &artifact.PackageArtifact{
					Runtime: s.ToPackage(),
				}

				art, err := mapArtifacts.MatchVersion(pMatch)
				if err != nil {
					fail = true
					errQueue = append(errQueue, err)
					continue
				}

				repoName := art.GetRepository()

				if repoName == "" {
					Warning(
						fmt.Sprintf("Unexpected repository string for package %s",
							s.GetName()))
					continue
				}

				// Create WagonRepository if present
				if _, ok := mapRepos[repoName]; !ok {

					repobasedir := config.GetSystem().GetRepoDatabaseDirPath(repoName)
					repo, err := config.GetSystemRepository(repoName)
					if err != nil {
						Error(
							fmt.Sprintf("Repository not found for stone %s",
								s.GetName()))
						fail = true
						continue
					}

					r := wagon.NewWagonRepository(repo)
					err = r.ReadWagonIdentify(repobasedir)
					if err != nil {
						fail = true
						Error("Error on read repository identity file: " + err.Error())
						continue
					}

					mapRepos[repoName] = r

				}

				pkgsQueue = append(pkgsQueue,
					&StonePolished{
						Stone:    s,
						Artifact: art,
					})

			}

			// Download all packages
			ndownloads := len(pkgsQueue)
			for idx, _ := range pkgsQueue {
				r := mapRepos[pkgsQueue[idx].Artifact.GetRepository()]
				a := pkgsQueue[idx].Artifact

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

			toFinalize := []*artifact.PackageArtifact{}

			nPkgs := len(pkgsQueue)
			for idx, _ := range pkgsQueue {
				r := mapRepos[pkgsQueue[idx].Artifact.GetRepository()]
				a := pkgsQueue[idx].Artifact
				s := pkgsQueue[idx].Stone

				repos := ""
				if a.GetPackage().Repository != "" {
					repos = "::" + a.GetPackage().Repository
				}
				msg := fmt.Sprintf(
					"[%3d of %3d] %-65s - %-15s",
					aurora.Bold(aurora.BrightMagenta(idx+1)),
					aurora.Bold(aurora.BrightMagenta(nPkgs)),
					fmt.Sprintf("%s%s", a.GetPackage().PackageName(),
						repos,
					),
					a.GetPackage().GetVersion())

				// When local database is broken could be with
				// empty list on array list. In this case, I using
				// from artifact the list if stone files list is empty.
				if len(s.Files) == 0 {
					s.Files = a.Files
				}

				err = aManager.ReinstallPackage(
					s, a, r, config.GetSystem().Rootfs,
					preserveSystem,
					force,
				)
				if err != nil {
					fail = true
					fmt.Println(fmt.Sprintf(
						"Error on reinstall package %s: %s",
						s.HumanReadableString(),
						err.Error()))
					Error(fmt.Sprintf(":package:%s # install failer :fire:", msg))

				} else {
					Info(fmt.Sprintf(":shortcake:%s # installed :check_mark:", msg))

					if !skipFinalizers {
						toFinalize = append(toFinalize, a)
					}
				}
			}

			if len(*stones) == 0 {
				Warning("No packages found.")
				fail = true
			} else if len(*stones) != len(pkgs) {
				Warning("Not all packages found.")
				fail = true
			}

			// Run finalizer of the installed packages
			if len(toFinalize) > 0 {
				for idx, _ := range toFinalize {
					r := mapRepos[toFinalize[idx].GetRepository()]
					err = aManager.ExecuteFinalizer(
						toFinalize[idx], r,
						true,
						config.GetSystem().Rootfs)
					if err != nil {
						fail = true
					}
				}
			}

			if len(errQueue) > 0 {
				for _, e := range errQueue {
					Warning(e)
				}
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

	flags.StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	flags.Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	flags.Bool("force", false, "Skip errors and force reinstall.")

	return ans
}

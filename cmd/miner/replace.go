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

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func NewReplacePackage(config *cfg.LuetConfig) *cobra.Command {
	var forArgs []string

	var ans = &cobra.Command{
		Use:     "replace-package <pkg1> --for <pkg2>,<pkgN>",
		Short:   `Replace a package without others packages in conflicts.`,
		Aliases: []string{"re", "switch"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("Missing arguments.")
				os.Exit(1)
			}
			if len(forArgs) < 1 {
				fmt.Println("Missing --for arguments.")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			pkgs := []*pkg.DefaultPackage{}
			forPkgs := []*pkg.DefaultPackage{}

			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			skipCheckSystem, _ := cmd.Flags().GetBool("skip-check-system")
			force, _ := cmd.Flags().GetBool("force")
			withDeps, _ := cmd.Flags().GetBool("with-deps")
			ignoreMasks, _ := cmd.Flags().GetBool("ignore-masks")

			InfoC(fmt.Sprintf(":rocket:%s %s",
				Bold(Blue("Luet")), Bold(Blue(util.Version()))))

			// Parse package to replace
			for _, pstr := range args {
				p, err := helpers.ParsePackageStr(config, pstr)
				if err != nil {
					Error(":firecracker:Error on parse package string " + pstr + ": " +
						err.Error())
					os.Exit(1)
				}

				pkgs = append(pkgs, p)
			}

			// Parse package to install
			for _, pstr := range forArgs {
				p, err := helpers.ParsePackageStr(config, pstr)
				if err != nil {
					Error(":firecracker:Error on parse package string " + pstr + ": " +
						err.Error())
					os.Exit(1)
				}

				forPkgs = append(forPkgs, p)
			}

			// Load finalizer runtime environments
			err := util.SetCliFinalizerEnvs(finalizerEnvs)
			if err != nil {
				Fatal(err.Error())
			}

			// Search for the installed packages
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

			// Load config protect configs
			installer.LoadConfigProtectConfs(config)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			if len(*stones) != len(args) {
				Error(":fire:Not all packages selected for the replacement are installed.")
				if len(*stones) == 0 {
					Info("No packages selected.")
				} else {
					Info("Matched packages are:")
					for _, s := range *stones {
						Info(s.HumanReadableString())
					}
				}
				os.Exit(1)
			}

			searchOpts.Packages = forPkgs

			// Searching the packages over the existing repos used
			// to replace the selected packages.
			reposArtifacts, err := searcher.SearchArtifacts(searchOpts)
			if err != nil {
				Error(err.Error())
				os.Exit(1)
			}

			if len(*reposArtifacts) != len(forPkgs) {
				Error(":fire:Not all packages defined for the replacement are availables.")
				if len(*reposArtifacts) == 0 {
					Info("No packages available for the replacements.")
				} else {
					Info("Missed packages:")
					aPack := &artifact.ArtifactsPack{
						Artifacts: (*reposArtifacts),
					}
					mapArtifacts := aPack.ToMap()
					for _, p := range forPkgs {
						if !mapArtifacts.HasKey(p.PackageName()) {
							Info(p.PackageName())
						}
					}
				}
				os.Exit(1)
			}
			reposArtifacts = nil

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			fail := false

			InfoC(":coffee:Removing selected packages...")
			// Before install all packages I remove all package
			// selected to avoid conflicts on resolver.
			nOpsTot := len(*stones)
			for idx, s := range *stones {
				repos := ""
				if s.Repository != "" {
					repos = "::" + s.Repository
				}

				msg := fmt.Sprintf(
					"[%3d of %3d] %-65s - %-15s",
					Bold(BrightMagenta(idx+1)),
					Bold(BrightMagenta(nOpsTot)),
					fmt.Sprintf("%s%s", s.GetName(),
						repos,
					),
					s.GetVersion())

				err = aManager.RemovePackage(
					s, config.GetSystem().Rootfs,
					preserveSystem,
					skipFinalizers,
					force,
				)
				if err != nil {
					fail = true
					fmt.Println(fmt.Sprintf(
						"Error on uninstall artifact %s: %s",
						s.HumanReadableString(),
						err.Error()))
					Error(fmt.Sprintf(":package:%s # uninstall failed - :fire:", msg))
					break
				} else {
					Info(fmt.Sprintf(":recycle: %s # uninstalled :check_mark:", msg))
				}

			}

			if fail {
				Error("Unexpected state. Operations stopped.")
				os.Exit(1)
			}

			opts := &installer.InstallOpts{
				Force:                       force,
				IgnoreConflicts:             true,
				NoDeps:                      !withDeps,
				PreserveSystemEssentialData: preserveSystem,
				Ask:                         false,
				SkipFinalizers:              skipFinalizers,
				Pretend:                     false,
				DownloadOnly:                false,
				CheckSystemFiles:            !skipCheckSystem,
				IgnoreMasks:                 ignoreMasks,
				ShowInstallOrder:            false,
				//IgnoreMasks:                 ignoreMasks,
			}

			if err := aManager.Install(opts, config.GetSystem().Rootfs,
				forPkgs...,
			); err != nil {
				Fatal("Error: " + err.Error())
			}

			InfoC(fmt.Sprintf(":confetti_ball:%s",
				Bold(Blue("All done."))))

		},
	}

	flags := ans.Flags()

	flags.String("system-dbpath", "", "System db path")
	flags.String("system-target", "", "System rootpath")
	flags.String("system-engine", "", "System DB engine")
	flags.Bool("preserve-system-essentials", true, "Preserve system essentials files.")
	flags.Bool("with-deps", false, "Consider package dependencies")
	flags.StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	flags.Bool("skip-check-system", false, "Skip conflicts check with existing rootfs.")
	flags.Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	flags.Bool("force", false, "Skip errors and force reinstall.")
	flags.Bool("ignore-masks", false, "Ignore packages masked.")
	flags.StringSliceVar(&forArgs, "for", []string{},
		"List of the package to install as replacement.")
	return ans
}

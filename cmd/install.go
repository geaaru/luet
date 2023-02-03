/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	helpers "github.com/geaaru/luet/cmd/helpers"
	cmdrepo "github.com/geaaru/luet/cmd/repo"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/subsets"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

func newInstallCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "install <pkg1> <pkg2> ...",
		Short: "Install a package",
		Long: `Installs one or more packages without asking questions:

	$ luet install -y utils/busybox utils/yq ...
	
To install only deps of a package:
	
	$ luet install --onlydeps utils/busybox ...
	
To not install deps of a package:
	
	$ luet install --nodeps utils/busybox ...

To force install a package:
	
	$ luet install --force utils/busybox ...
`,
		Aliases: []string{"i"},
		PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			config.Viper.BindPFlag("nodeps", cmd.Flags().Lookup("nodeps"))
			config.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			config.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
			config.Viper.BindPFlag("general.overwrite_dir_perms",
				cmd.Flags().Lookup("Overwrite exiting directories permissions."))
		},
		Run: func(cmd *cobra.Command, args []string) {
			var toInstall pkg.DefaultPackages

			InfoC(fmt.Sprintf(":rocket:%s %s",
				Bold(Blue("Luet")), Bold(Blue(util.Version()))))

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toInstall = append(toInstall, pack)
			}

			force := config.Viper.GetBool("force")
			nodeps := config.Viper.GetBool("nodeps")
			//onlydeps := config.Viper.GetBool("onlydeps")
			yes := config.Viper.GetBool("yes")
			pretend, _ := cmd.Flags().GetBool("pretend")
			skipCheckSystem, _ := cmd.Flags().GetBool("skip-check-system")
			ignoreConflicts, _ := cmd.Flags().GetBool("ignore-conflicts")
			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")
			downloadOnly, _ := cmd.Flags().GetBool("download-only")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")

			if syncRepos {
				waitGroup := &sync.WaitGroup{}
				sem := semaphore.NewWeighted(int64(config.GetGeneral().Concurrency))
				ctx := context.TODO()

				defer waitGroup.Wait()

				var ch chan util.ChannelRepoOpRes = make(
					chan util.ChannelRepoOpRes,
					config.GetGeneral().Concurrency,
				)
				// Using new way
				nOps := 0

				for idx, repo := range config.SystemRepositories {
					if repo.Enable {
						waitGroup.Add(1)
						go cmdrepo.ProcessRepository(
							&config.SystemRepositories[idx], config, ch, force, sem,
							waitGroup, &ctx)
						nOps++
					}
				}

				res := 0
				if nOps > 0 {
					for i := 0; i < nOps; i++ {
						resp := <-ch
						if resp.Error != nil && !force {
							res = 1
							Error("Error on update repository " + resp.Repo.Name + ": " +
								resp.Error.Error())
						}
					}
				} else {
					fmt.Println("No repositories candidates found.")
				}

				if res != 0 {
					os.Exit(res)
				}

			}

			// Load config protect configs
			installer.LoadConfigProtectConfs(config)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			// Load finalizer runtime environments
			err := util.SetCliFinalizerEnvs(finalizerEnvs)
			if err != nil {
				Fatal(err.Error())
			}

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			opts := &installer.InstallOpts{
				Force:                       force,
				IgnoreConflicts:             ignoreConflicts,
				NoDeps:                      nodeps,
				PreserveSystemEssentialData: preserveSystem,
				Ask:                         !yes,
				SkipFinalizers:              skipFinalizers,
				Pretend:                     pretend,
				DownloadOnly:                downloadOnly,
				CheckSystemFiles:            !skipCheckSystem,
			}

			if err := aManager.Install(opts, config.GetSystem().Rootfs,
				toInstall...,
			); err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	flags := ans.Flags()

	flags.Bool("nodeps", false, "Don't consider package dependencies (harmful!)")
	flags.Bool("ignore-conflicts", false, "Don't consider package conflicts (harmful!)")
	flags.Bool("skip-check-system", false, "Skip conflicts check with existing rootfs.")
	flags.BoolP("pretend", "p", false,
		"simply display what *would* have been installed if --pretend weren't used")

	//flags.Bool("onlydeps", false, "Consider **only** package dependencies")
	flags.Bool("force", false, "Skip errors and keep going (potentially harmful)")
	flags.Bool("preserve-system-essentials", true, "Preserve system luet files")
	flags.BoolP("yes", "y", false, "Don't ask questions")
	flags.Bool("download-only", false, "Download only")
	flags.StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	flags.Bool("overwrite-existing-dir-perms", false,
		"Overwrite exiting directories permissions.")
	flags.Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	flags.Bool("sync-repos", false,
		"Sync repositories before install. Note: If there are in memory repositories then the sync is done always.")

	return ans
}

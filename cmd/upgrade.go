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

	cmdrepo "github.com/geaaru/luet/cmd/repo"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

func newUpgradeCommand(config *cfg.LuetConfig) *cobra.Command {

	var upgradeCmd = &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrades the system",
		Aliases: []string{"u"},
		PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			config.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
		},
		Long: `Upgrades packages installed.`,
		Run: func(cmd *cobra.Command, args []string) {

			InfoC(fmt.Sprintf(":rocket:%s %s",
				Bold(Blue("Luet")), Bold(Blue(util.Version()))))

			force := config.Viper.GetBool("force")
			nodeps, _ := cmd.Flags().GetBool("nodeps")
			yes := config.Viper.GetBool("yes")
			skipCheckSystem, _ := cmd.Flags().GetBool("skip-check-system")
			downloadOnly, _ := cmd.Flags().GetBool("download-only")
			pretend, _ := cmd.Flags().GetBool("pretend")
			ignoreConflicts, _ := cmd.Flags().GetBool("ignore-conflicts")
			preserveSystem, _ := cmd.Flags().GetBool("preserve-system-essentials")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")
			ignoreMasks, _ := cmd.Flags().GetBool("ignore-masks")
			showUpgradeOrder, _ := cmd.Flags().GetBool("show-upgrade-order")

			// TODO: Move this inside the ArtifactManager or
			//       to a common function.
			if syncRepos {
				waitGroup := &sync.WaitGroup{}
				sem := semaphore.NewWeighted(int64(config.GetGeneral().Concurrency))
				ctx := context.TODO()

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

				waitGroup.Wait()

				if res != 0 {
					os.Exit(res)
				}

				waitGroup = nil
				ch = nil
				sem = nil
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
				IgnoreMasks:                 ignoreMasks,
				ShowInstallOrder:            showUpgradeOrder,
			}
			if err := aManager.Upgrade(opts, config.GetSystem().Rootfs); err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	flags := upgradeCmd.Flags()

	flags.BoolP("pretend", "p", false,
		"simply display what *would* have been upgraded if --pretend weren't used")
	flags.Bool("nodeps", false, "Don't consider package dependencies (harmful!)")
	flags.Bool("ignore-conflicts", false, "Don't consider package conflicts (harmful!)")
	flags.Bool("skip-check-system", false, "Skip conflicts check with existing rootfs.")
	flags.Bool("force", false, "Force upgrade by ignoring errors")
	flags.StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	flags.Bool("preserve-system-essentials", true, "Preserve system luet files")
	flags.BoolP("yes", "y", false, "Don't ask questions")
	flags.Bool("download-only", false, "Download only")
	flags.Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	flags.Bool("sync-repos", false,
		"Sync repositories before upgrade. Note: If there are in memory repositories then the sync is done always.")
	flags.Bool("ignore-masks", false, "Ignore packages masked.")
	flags.Bool("show-upgrade-order", false,
		"In additional of the package to upgrade show the installation order and exit.")

	return upgradeCmd
}

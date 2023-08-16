/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"
	installer "github.com/geaaru/luet/pkg/v2/installer"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func newUpgradeCommand(config *cfg.LuetConfig) *cobra.Command {

	var upgradeCmd = &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrades Luet package",
		Aliases: []string{"u"},
		Long:    `Upgrades packages installed.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			config.Viper.BindPFlag("nodeps", cmd.Flags().Lookup("nodeps"))
			config.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			config.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
		},
		Run: func(cmd *cobra.Command, args []string) {

			InfoC(fmt.Sprintf(":rocket:%s %s",
				Bold(Blue("Luet")), Bold(Blue(util.Version()))))

			force := config.Viper.GetBool("force")
			nodeps := config.Viper.GetBool("nodeps")
			yes := config.Viper.GetBool("yes")
			//force, _ := cmd.Flags().GetBool("force")
			//nodeps, _ := cmd.Flags().GetBool("nodeps")
			//yes, _ := cmd.Flags().GetBool("yes")
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
			deep, _ := cmd.Flags().GetBool("deep")
			purge, _ := cmd.Flags().GetBool("purge-repos")
			cleanup, _ := cmd.Flags().GetBool("cleanup")

			if syncRepos {
				optsRails := &wagon.SyncOpts{
					Force:        force,
					IgnoreErrors: false,
				}
				rails := wagon.NewWagonsRails(config)
				err := rails.SyncRepos([]string{}, optsRails)
				if err != nil {
					Error(err.Error())
					os.Exit(1)
				}
				rails = nil
				optsRails = nil
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
				Deep:                        deep,
			}
			if err := aManager.Upgrade(opts, config.GetSystem().Rootfs); err != nil {
				Fatal("Error: " + err.Error())
			}

			if cleanup {
				err = aManager.CleanLocalPackagesCache()
				if err != nil {
					Fatal(err.Error())
				}

				if purge {
					err = aManager.PurgeLocalReposCache()
					if err != nil {
						Fatal(err.Error())
					}
				}
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
	flags.Bool("deep", false, "Deep analyzing with downgrade.")
	flags.Bool("download-only", false, "Download only")
	flags.Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	flags.Bool("sync-repos", false,
		"Sync repositories before upgrade. Note: If there are in memory repositories then the sync is done always.")
	flags.Bool("ignore-masks", false, "Ignore packages masked.")
	flags.Bool("show-upgrade-order", false,
		"In additional of the package to upgrade show the installation order and exit.")
	flags.Bool("cleanup", false, "Cleanup local packages cache.")
	flags.Bool("purge-repos", false,
		"Remove all repos files. This impacts on searching packages too. Needs --cleanup.")

	return upgradeCmd
}

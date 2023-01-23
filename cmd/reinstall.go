// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
//	Daniele Rondina <geaaru@sabayonlinux.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.
package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	helpers "github.com/geaaru/luet/cmd/helpers"
	cmdrepo "github.com/geaaru/luet/cmd/repo"
	"github.com/geaaru/luet/cmd/util"
	config "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
	"github.com/geaaru/luet/pkg/subsets"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

func newReinstallCommand(cfg *config.LuetConfig) *cobra.Command {

	var reinstallCmd = &cobra.Command{
		Use:   "reinstall <pkg1> <pkg2> <pkg3>",
		Short: "reinstall a set of packages",
		Long: `Reinstall a group of packages in the system:

		$ luet reinstall -y system/busybox shells/bash system/coreutils ...
	`,
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSolverFlags(cmd)
			cfg.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			cfg.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			cfg.Viper.BindPFlag("for", cmd.Flags().Lookup("for"))
			cfg.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			var toUninstall pkg.Packages
			var toAdd pkg.Packages

			force := cfg.Viper.GetBool("force")
			onlydeps := cfg.Viper.GetBool("onlydeps")
			yes := cfg.Viper.GetBool("yes")

			downloadOnly, _ := cmd.Flags().GetBool("download-only")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")

			Info("Luet version", util.Version())

			if syncRepos {

				waitGroup := &sync.WaitGroup{}
				sem := semaphore.NewWeighted(int64(cfg.GetGeneral().Concurrency))
				ctx := context.TODO()

				defer waitGroup.Wait()

				var ch chan util.ChannelRepoOpRes = make(
					chan util.ChannelRepoOpRes,
					cfg.GetGeneral().Concurrency,
				)
				// Using new way
				nOps := 0

				for idx, repo := range cfg.SystemRepositories {
					if repo.Enable {
						waitGroup.Add(1)
						go cmdrepo.ProcessRepository(&cfg.SystemRepositories[idx], cfg, ch, force, sem, waitGroup, &ctx)
						nOps++
					}
				}

				res := 0
				if nOps > 0 {
					for i := 0; i < nOps; i++ {
						resp := <-ch
						if resp.Error != nil && !force {
							res = 1
							Error("Error on update repository " + resp.Repo.Name + ": " + resp.Error.Error())
						}
					}
				} else {
					fmt.Println("No repositories candidates found.")
				}

				if res != 0 {
					os.Exit(res)
				}

			}

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toUninstall = append(toUninstall, pack)
				toAdd = append(toAdd, pack)
			}

			// This shouldn't be necessary, but we need to unmarshal the repositories to a concrete struct, thus we need to port them back to the Repositories type
			repos := installer.Repositories{}
			for _, repo := range cfg.SystemRepositories {
				if !repo.Enable {
					continue
				}
				r := installer.NewSystemRepository(repo)
				repos = append(repos, r)
			}

			util.SetSolverConfig()

			Debug("Solver", cfg.GetSolverOptions().CompactString())

			// Load config protect configs
			installer.LoadConfigProtectConfs(cfg)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(cfg)
			// Load subsets config
			subsets.LoadSubsetsConfig(cfg)

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
				Concurrency:                 cfg.GetGeneral().Concurrency,
				SolverOptions:               *cfg.GetSolverOptions(),
				NoDeps:                      true,
				Force:                       force,
				OnlyDeps:                    onlydeps,
				PreserveSystemEssentialData: true,
				Ask:                         !yes,
				DownloadOnly:                downloadOnly,
				SyncRepositories:            false,
			})
			inst.Repositories(repos)

			system := &installer.System{
				Database: cfg.GetSystemDB(),
				Target:   cfg.GetSystem().Rootfs,
			}
			err := inst.Swap(toUninstall, toAdd, system)
			if err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	reinstallCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	reinstallCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	reinstallCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	reinstallCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	reinstallCmd.Flags().Bool("onlydeps", false, "Consider **only** package dependencies")
	reinstallCmd.Flags().Bool("force", false, "Skip errors and keep going (potentially harmful)")
	reinstallCmd.Flags().Bool("solver-concurrent", false, "Use concurrent solver (experimental)")
	reinstallCmd.Flags().BoolP("yes", "y", false, "Don't ask questions")
	reinstallCmd.Flags().Bool("download-only", false, "Download only")
	reinstallCmd.Flags().Bool("sync-repos", false,
		"Sync repositories before install. Note: If there are in memory repositories then the sync is done always.")
	return reinstallCmd
}

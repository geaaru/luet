// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
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

	cmdrepo "github.com/geaaru/luet/cmd/repo"
	"github.com/geaaru/luet/cmd/util"
	config "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

func newReclaimCommand(cfg *config.LuetConfig) *cobra.Command {

	var reclaimCmd = &cobra.Command{
		Use:   "reclaim",
		Short: "Reclaim packages to Luet database from available repositories",
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSystemFlags(cmd)
			cfg.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
		},
		Long: `Reclaim tries to find association between packages in the online repositories and the system one.

		$ luet reclaim

	It scans the target file system, and if finds a match with a package available in the repositories, it marks as installed in the system database.
	`,
		Run: func(cmd *cobra.Command, args []string) {
			// This shouldn't be necessary, but we need to unmarshal the repositories to a concrete struct, thus we need to port them back to the Repositories type
			repos := installer.Repositories{}
			for _, repo := range cfg.SystemRepositories {
				if !repo.Enable {
					continue
				}
				r := installer.NewSystemRepository(repo)
				repos = append(repos, r)
			}

			force := cfg.Viper.GetBool("force")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")

			Debug("Solver", cfg.GetSolverOptions().CompactString())

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

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
				Concurrency:                 cfg.GetGeneral().Concurrency,
				Force:                       force,
				PreserveSystemEssentialData: true,
				SyncRepositories:            false,
			})
			inst.Repositories(repos)

			system := &installer.System{
				Database: cfg.GetSystemDB(),
				Target:   cfg.GetSystem().Rootfs,
			}
			err := inst.Reclaim(system)
			if err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	reclaimCmd.Flags().Bool("force", false, "Skip errors and keep going (potentially harmful)")

	reclaimCmd.Flags().Bool("sync-repos", false,
		"Sync repositories before reclaim. Note: If there are in memory repositories then the sync is done always.")
	return reclaimCmd
}

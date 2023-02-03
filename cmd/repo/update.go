// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//                  Daniele Rondina <geaaru@sabayonlinux.org>
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

package cmd_repo

import (
	"context"
	"fmt"
	"os"
	"sync"

	cmd_util "github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/cobra"
)

func ProcessRepository(repo *cfg.LuetRepository, config *cfg.LuetConfig,
	channel chan cmd_util.ChannelRepoOpRes, force bool,
	sem *semaphore.Weighted, waitGroup *sync.WaitGroup, ctx *context.Context) {

	repobasedir := config.GetSystem().GetRepoDatabaseDirPath(repo.Name)

	defer waitGroup.Done()

	err := sem.Acquire(*ctx, 1)
	if err != nil {
		return
	}
	defer sem.Release(1)

	r := wagon.NewWagonRepository(repo)
	if r.HasLocalWagonIdentity(repobasedir) {
		err = r.ReadWagonIdentify(repobasedir)
		if err != nil && (!force) {
			channel <- cmd_util.ChannelRepoOpRes{err, repo}
			return
		}
	}

	err = r.Sync(force)
	r.ClearCatalog()
	r = nil

	if err != nil {
		channel <- cmd_util.ChannelRepoOpRes{err, repo}
	} else {
		channel <- cmd_util.ChannelRepoOpRes{nil, repo}
	}
	return
}

func NewRepoUpdateCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "update [repo1] [repo2] [OPTIONS]",
		Short: "Update a specific cached repository or all cached repositories.",
		Example: `
# Update all cached repositories:
$> luet repo update

# Update only repo1 and repo2
$> luet repo update repo1 repo2
`,
		Aliases: []string{"up"},
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		Run: func(cmd *cobra.Command, args []string) {

			ignore, _ := cmd.Flags().GetBool("ignore-errors")
			force, _ := cmd.Flags().GetBool("force")
			nOps := 0
			var ch chan cmd_util.ChannelRepoOpRes = make(
				chan cmd_util.ChannelRepoOpRes,
				config.GetGeneral().Concurrency,
			)

			waitGroup := &sync.WaitGroup{}
			sem := semaphore.NewWeighted(int64(config.GetGeneral().Concurrency))
			ctx := context.TODO()

			if len(args) > 0 {
				for _, rname := range args {
					repo, err := config.GetSystemRepository(rname)
					if err != nil && !ignore {
						Fatal(err.Error())
					} else if err != nil {
						continue
					}
					waitGroup.Add(1)

					go ProcessRepository(repo, config, ch, force, sem, waitGroup, &ctx)
					nOps++
				}

			} else {
				for idx, repo := range config.SystemRepositories {
					if repo.Enable {
						waitGroup.Add(1)
						go ProcessRepository(&config.SystemRepositories[idx], config, ch, force, sem, waitGroup, &ctx)
						nOps++
					}
				}
			}

			res := 0
			if nOps > 0 {
				for i := 0; i < nOps; i++ {
					resp := <-ch
					if resp.Error != nil && !ignore {
						res = 1
						Error("Error on update repository " + resp.Repo.Name + ": " + resp.Error.Error())
					}
				}
			} else {
				fmt.Println("No repositories candidates found.")
			}

			waitGroup.Wait()

			os.Exit(res)
		},
	}

	ans.Flags().BoolP("ignore-errors", "i", false, "Ignore errors on sync repositories.")
	ans.Flags().BoolP("force", "f", false, "Force resync.")

	return ans
}

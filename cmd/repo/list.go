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
	"fmt"
	"strconv"
	"time"

	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func NewRepoListCommand() *cobra.Command {
	var ans = &cobra.Command{
		Use:   "list [OPTIONS]",
		Short: "List of the configured repositories.",
		Args:  cobra.OnlyValidArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		Run: func(cmd *cobra.Command, args []string) {
			var repoColor, repoText, repoRevision string

			enable, _ := cmd.Flags().GetBool("enabled")
			quiet, _ := cmd.Flags().GetBool("quiet")
			repoType, _ := cmd.Flags().GetString("type")

			for idx, _ := range LuetCfg.SystemRepositories {
				repo := LuetCfg.SystemRepositories[idx]
				if enable && !repo.Enable {
					continue
				}

				if repoType != "" && repo.Type != repoType {
					continue
				}

				repoRevision = ""

				if quiet {
					fmt.Println(repo.Name)
				} else {
					if repo.Enable {
						repoColor = Bold(BrightGreen(repo.Name)).String()
					} else {
						repoColor = Bold(BrightRed(repo.Name)).String()
					}
					if repo.Description != "" {
						repoText = Yellow(repo.Description).String()
					} else {
						repoText = Yellow(repo.Urls[0]).String()
					}

					repobasedir := LuetCfg.GetSystem().GetRepoDatabaseDirPath(repo.Name)
					if repo.Cached {

						r := wagon.NewWagonRepository(&repo)
						err := r.ReadWagonIdentify(repobasedir)
						if err != nil {
							Warning("Error on read repository identity file: " + err.Error())
						} else {
							tsec, _ := strconv.ParseInt(r.GetLastUpdate(), 10, 64)
							repoRevision = Bold(Red(r.GetRevision())).String() +
								" - " + Bold(Green(time.Unix(tsec, 0).String())).String()
						}
					}

					if repoRevision != "" {
						fmt.Println(
							fmt.Sprintf("%s\n  %s\n  Revision %s", repoColor, repoText, repoRevision))
					} else {
						fmt.Println(
							fmt.Sprintf("%s\n  %s", repoColor, repoText))
					}
				}
			}
		},
	}

	ans.Flags().Bool("enabled", false, "Show only enable repositories.")
	ans.Flags().BoolP("quiet", "q", false, "Show only name of the repositories.")
	ans.Flags().StringP("type", "t", "", "Filter repositories of a specific type")

	return ans
}

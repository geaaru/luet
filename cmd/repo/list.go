/*
Copyright © 2021-2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_repo

import (
	"fmt"
	"os"
	"strconv"
	"time"

	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func NewRepoListCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "list [OPTIONS]",
		Short: "List of the configured repositories.",
		Args:  cobra.OnlyValidArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			enable, _ := cmd.Flags().GetBool("enabled")
			disable, _ := cmd.Flags().GetBool("disable")
			if enable && disable {
				Error(
					"Used both --enable and --disabled options.\nOnly one admitted.",
				)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			var repoColor, repoText, repoRevision string

			enable, _ := cmd.Flags().GetBool("enabled")
			disable, _ := cmd.Flags().GetBool("disabled")
			quiet, _ := cmd.Flags().GetBool("quiet")
			repoType, _ := cmd.Flags().GetString("type")

			for idx, _ := range config.SystemRepositories {
				repo := config.SystemRepositories[idx]
				if enable && !repo.Enable {
					continue
				}
				if disable && repo.Enable {
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

					repobasedir := config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
					if repo.Cached {

						r := wagon.NewWagonRepository(&repo)
						if r.HasLocalWagonIdentity(repobasedir) {
							err := r.ReadWagonIdentify(repobasedir)
							if err != nil {
								Warning("Error on read repository identity file: " + err.Error())
							} else {
								tsec, _ := strconv.ParseInt(r.GetLastUpdate(), 10, 64)
								repoRevision = fmt.Sprintf(
									"%s - %s",
									Bold(Red(fmt.Sprintf("%5d", r.GetRevision()))).String(),
									Bold(Green(time.Unix(tsec, 0).String())).String())
							}
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

	ans.Flags().Bool("enabled", false, "Show only enabled repositories.")
	ans.Flags().Bool("disabled", false, "Show only disabled repositories.")
	ans.Flags().BoolP("quiet", "q", false, "Show only name of the repositories.")
	ans.Flags().StringP("type", "t", "", "Filter repositories of a specific type")

	return ans
}

/*
Copyright © 2021-2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_repo

import (
	"fmt"
	"io/ioutil"
	"os"

	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewRepoDisableCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "disable <repo1> ... <repoN>",
		Short: "Disable one or more repositories.",
		Args:  cobra.OnlyValidArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Missing repositories to disable.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			res := 0
			for _, r := range args {

				repo, err := config.GetSystemRepository(r)
				if err != nil {
					res = 1
					Warning(fmt.Sprintf(
						"Error on retrieve repository with name %s: %s",
						r, err.Error()))
					continue
				}

				if !repo.Enable {
					fmt.Println(fmt.Sprintf(
						"Repository %s already disabled. Nothing to do.",
						r,
					))
					continue
				}

				rNew := repo.Clone()
				rNew.Verify = false
				rNew.Revision = 0
				rNew.LastUpdate = ""
				rNew.Enable = false

				data, err := yaml.Marshal(rNew)
				if err != nil {
					Error(fmt.Sprintf(
						"Error on marshal repository %s: %s",
						r, err.Error()))
					res = 1
					continue
				}

				err = ioutil.WriteFile(repo.File, data, os.ModePerm)
				if err != nil {
					Error(fmt.Sprintf(
						"Error on disable repository %s: %s",
						r, err.Error()))
					res = 1
					continue
				}

				InfoC(fmt.Sprintf("%s disabled: :heavy_check_mark:",
					Bold(BrightRed(repo.Name))))
			}

			os.Exit(res)
		},
	}

	return ans
}

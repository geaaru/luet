// Copyright © 2021 Daniele Rondina <geaaru@funtoo.org>
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

package cmd_query

import (
	"encoding/json"
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewQueryFilesCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "files <pkg1> ... <pkgN> [OPTIONS]",
		Short:   "Show files owned by a specific package.",
		Aliases: []string{"fi", "f"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Missing package")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			out, _ := cmd.Flags().GetString("output")

			util.SetSystemConfig()
			util.SetSolverConfig()

			searchOpts := &wagon.StonesSearchOpts{
				Categories:    []string{},
				Labels:        []string{},
				LabelsMatches: []string{},
				Matches:       []string{},
				Hidden:        true,
				AndCondition:  false,
				WithFiles:     true,
			}

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				searchOpts.Packages = append(searchOpts.Packages, pack)
			}

			config.GetLogging().SetLogLevel("error")

			res, err := util.SearchFromRepos(config, searchOpts)
			if err != nil {
				Fatal("Error on retrieve packages ", err.Error())
			}

			if out != "yaml" && out != "json" {
				for _, s := range *res {
					for _, f := range s.Files {
						fmt.Println(f)
					}
				}
			} else {
				ans := []string{}
				for _, s := range *res {
					ans = append(ans, s.Files...)
				}

				switch out {
				case "json":
					data, err := json.Marshal(ans)
					if err != nil {
						Fatal("Error on marshal data ", err.Error())
					}
					fmt.Println(string(data))
				default:
					data, err := yaml.Marshal(ans)
					if err != nil {
						Fatal("Error on marshal data ", err.Error())
					}
					fmt.Println(string(data))
				}
			}

		},
	}

	ans.Flags().StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	return ans
}

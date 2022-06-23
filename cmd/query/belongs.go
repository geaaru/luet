// Copyright © 2021-2022 Daniele Rondina <geaaru@funtoo.org>
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
	"strings"

	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewQueryBelongsCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "belongs <file1> ... <fileN> [OPTIONS]",
		Short:   "Resolve what package a file belongs to.",
		Aliases: []string{"be", "b"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Missing package")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			out, _ := cmd.Flags().GetString("output")
			tableMode, _ := cmd.Flags().GetBool("table")
			quiet, _ := cmd.Flags().GetBool("quiet")

			util.SetSystemConfig()
			util.SetSolverConfig()

			// Files inside the metadata are store without initial /
			// I drop it if defined.
			files := []string{}
			for _, s := range args {
				if strings.HasPrefix(s, "/") {
					files = append(files, s[1:])
				} else {
					files = append(files, s)
				}
			}

			if !config.GetGeneral().Debug {
				config.GetLogging().SetLogLevel("error")
			}

			searchOpts := &wagon.StonesSearchOpts{
				Categories:    []string{},
				Labels:        []string{},
				LabelsMatches: []string{},
				Matches:       []string{},
				FilesOwner:    files,
				Hidden:        true,
				AndCondition:  false,
				WithFiles:     true,
			}

			res, err := util.SearchFromRepos(config, searchOpts)
			if err != nil {
				Fatal("Error on retrieve packages ", err.Error())
			}

			if out == "json" {
				pack := wagon.StonesPack{*res}
				data, err := json.Marshal(pack)
				if err != nil {
					fmt.Println("Error on marshal stones ", err.Error())
					os.Exit(1)
				}
				fmt.Println(string(data))
			} else if out == "yaml" {
				pack := wagon.StonesPack{*res}
				data, err := yaml.Marshal(pack)
				if err != nil {
					fmt.Println("Error on marshal stones ", err.Error())
					os.Exit(1)
				}
				fmt.Println(string(data))
			} else {

				if tableMode {

					table := tablewriter.NewWriter(os.Stdout)
					table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
					table.SetCenterSeparator("|")
					table.SetAlignment(tablewriter.ALIGN_LEFT)
					table.SetHeader([]string{
						"Package", "Version", "Repository",
					})
					table.SetAutoWrapText(false)

					for _, s := range *res {
						table.Append([]string{
							fmt.Sprintf("%s/%s", s.Category, s.Name),
							s.Version,
							s.Repository,
						})
					}

					table.Render()
				} else {
					for _, s := range *res {
						if quiet {
							fmt.Println(fmt.Sprintf("%s/%s", s.Category, s.Name))
						} else {
							fmt.Println(fmt.Sprintf("%s/%s-%s", s.Category, s.Name, s.Version))
						}
					}
				}
			}

		},
	}

	ans.Flags().StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	ans.Flags().Bool("table", false, "show output in a table (wider screens)")
	ans.Flags().Bool("quiet", false, "show output as list without version")

	return ans
}

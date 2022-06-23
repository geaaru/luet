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
	"encoding/json"
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/solver"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	tablewriter "github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newSearchCommand(config *cfg.LuetConfig) *cobra.Command {

	var labels []string
	var regLabels []string
	var categories []string
	var annotations []string
	var packages []string

	var searchCmd = &cobra.Command{
		Use:   "search <term>",
		Short: "Search packages",
		Long: `Search for installed and available packages
		
	To search a package in the repositories:

		$ luet search <regex1> ... <regexN>

	To search a package and display results in a table (wide screens):

		$ luet search --table <regex>

	To look into the installed packages:

		$ luet search --installed <regex>

	Note: the regex argument is optional, if omitted implies "all"

	To search a package by label:

		$ luet search --label <label1>,<label2>...,<labelN>

	or by regex against the label:

		$ luet search --rlabel <regex-label1>,..,<regex-labelN>

	or by categories:

		$ luet search --category <cat1>,..,<catN>

	or by annotations:

		$ luet search --annotation <annotation1>,..,<annotationN>

	or by package (used only category and package name for name in the format cat/foo)

	  $ luet search -p <cat/foo>,<cat/foo2>

	Search can also return results in the terminal in different ways: as terminal output, as json or as yaml.

		$ luet search -o json <regex> # JSON output
		$ luet search -o yaml <regex> # YAML output
	`,
		Aliases: []string{"s"},
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSystemFlags(cmd)
			util.BindSolverFlags(cmd)
			config.Viper.BindPFlag("installed", cmd.Flags().Lookup("installed"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			//var results Results
			if len(args) == 0 && len(packages) == 0 {
				args = []string{"."}
			}
			hidden, _ := cmd.Flags().GetBool("hidden")
			files, _ := cmd.Flags().GetBool("files")
			andCond, _ := cmd.Flags().GetBool("and-condition")
			installed := config.Viper.GetBool("installed")
			tableMode, _ := cmd.Flags().GetBool("table")
			quiet, _ := cmd.Flags().GetBool("quiet")

			util.SetSystemConfig()
			util.SetSolverConfig()

			out, _ := cmd.Flags().GetString("output")
			config.GetLogging().SetLogLevel("error")

			searchOpts := &wagon.StonesSearchOpts{
				Categories:    categories,
				Labels:        labels,
				LabelsMatches: regLabels,
				Matches:       args,
				Hidden:        hidden,
				AndCondition:  andCond,
				WithFiles:     files,
			}
			var res *[]*wagon.Stone
			var err error

			if len(packages) > 0 {
				for _, p := range packages {
					pack, err := helpers.ParsePackageStr(p)
					if err != nil {
						Fatal("Invalid package string ", p, ": ", err.Error())
					}
					searchOpts.Packages = append(searchOpts.Packages, pack)
				}
			}

			if installed {
				res, err = util.SearchInstalled(config, searchOpts)
				if err != nil {
					fmt.Println("Error on retrieve installed packages ", err.Error())
					os.Exit(1)
				}
			} else {
				res, err = util.SearchFromRepos(config, searchOpts)
				if err != nil {
					fmt.Println("Error on retrieve installed packages ", err.Error())
					os.Exit(1)
				}
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

	searchCmd.Flags().String("system-dbpath", "", "System db path")
	searchCmd.Flags().String("system-target", "", "System rootpath")
	searchCmd.Flags().String("system-engine", "", "System DB engine")
	searchCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	searchCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	searchCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	searchCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")

	searchCmd.Flags().Bool("installed", false, "Search between system packages")

	searchCmd.Flags().StringSliceVar(&labels, "label", []string{},
		"Search packages through one or more labels.")
	searchCmd.Flags().StringSliceVar(&regLabels, "rlabel", []string{},
		"Search packages through one or more labels regex.")
	searchCmd.Flags().StringSliceVar(&categories, "category", []string{},
		"Search packages through one or more categories regex.")
	searchCmd.Flags().StringSliceVarP(&annotations, "annotation", "a", []string{},
		"Search packages through one or more annotations.")
	searchCmd.Flags().StringSliceVarP(&packages, "package", "p", []string{},
		"Search packages matching the package string cat/name.")
	searchCmd.Flags().Bool("condition-and", false,
		"The searching options are managed in AND between the searching types.")

	searchCmd.Flags().StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")
	searchCmd.Flags().Bool("hidden", false, "Include hidden packages")
	searchCmd.Flags().Bool("files", false, "Show package files on YAML/JSON output.")
	searchCmd.Flags().Bool("table", false, "show output in a table (wider screens)")
	searchCmd.Flags().Bool("quiet", false, "show output as list without version")

	return searchCmd
}

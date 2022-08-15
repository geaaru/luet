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

	var ans = &cobra.Command{
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
		Run: func(cmd *cobra.Command, args []string) {
			//var results Results
			if len(args) == 0 && len(packages) == 0 {
				args = []string{"."}
			}
			hidden, _ := cmd.Flags().GetBool("hidden")
			files, _ := cmd.Flags().GetBool("files")
			orCond, _ := cmd.Flags().GetBool("condition-or")
			installed, _ := cmd.Flags().GetBool("installed")
			tableMode, _ := cmd.Flags().GetBool("table")
			quiet, _ := cmd.Flags().GetBool("quiet")
			mode2, _ := cmd.Flags().GetBool("mode2")
			full, _ := cmd.Flags().GetBool("full")

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
				AndCondition:  !orCond,
				WithFiles:     files,
				Modev2:        mode2,
				Full:          full,
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

	flags := ans.Flags()

	flags.String("system-dbpath", "", "System db path")
	flags.String("system-target", "", "System rootpath")
	flags.String("system-engine", "", "System DB engine")
	flags.String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	flags.Float32("solver-rate", 0.7, "Solver learning rate")
	flags.Float32("solver-discount", 1.0, "Solver discount rate")
	flags.Int("solver-attempts", 9000, "Solver maximum attempts")

	flags.Bool("installed", false, "Search between system packages")

	flags.StringSliceVar(&labels, "label", []string{},
		"Search packages through one or more labels.")
	flags.StringSliceVar(&regLabels, "rlabel", []string{},
		"Search packages through one or more labels regex.")
	flags.StringSliceVar(&categories, "category", []string{},
		"Search packages through one or more categories regex.")
	flags.StringSliceVarP(&annotations, "annotation", "a", []string{},
		"Search packages through one or more annotations.")
	flags.StringSliceVarP(&packages, "package", "p", []string{},
		"Search packages matching the package string cat/name.")
	flags.Bool("condition-or", false,
		"The searching options are managed in OR between the searching types.")

	flags.StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	flags.Bool("hidden", false, "Include hidden packages")
	flags.Bool("files", false, "Show package files on YAML/JSON output.")
	flags.Bool("table", false, "show output in a table (wider screens)")
	flags.Bool("quiet", false, "show output as list without version")
	flags.Bool("full", false, "Show full informations.")

	flags.Bool("mode2", true, "Using searching v2.")

	return ans
}

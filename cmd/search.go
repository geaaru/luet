/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	art "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	mask "github.com/geaaru/luet/pkg/v2/repository/mask"

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
	var names []string

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

	or by names:

		$ luet search --name|-n <name1>,..,<nameN>

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
			artifactView, _ := cmd.Flags().GetBool("artifacts")
			installed, _ := cmd.Flags().GetBool("installed")
			if installed && artifactView {
				fmt.Println(
					"Flags --installed and --artifacts not usable together.")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			//var results Results
			if len(args) == 0 && len(packages) == 0 {
				args = []string{"."}
			}
			hidden, _ := cmd.Flags().GetBool("hidden")
			files, _ := cmd.Flags().GetBool("files")
			withRootfsPrefix, _ := cmd.Flags().GetBool("with-rootfs-prefix")
			orCond, _ := cmd.Flags().GetBool("condition-or")
			installed, _ := cmd.Flags().GetBool("installed")
			tableMode, _ := cmd.Flags().GetBool("table")
			quiet, _ := cmd.Flags().GetBool("quiet")
			full, _ := cmd.Flags().GetBool("full")
			ignoreMasks, _ := cmd.Flags().GetBool("ignore-masks")
			artifactView, _ := cmd.Flags().GetBool("artifacts")

			util.SetSystemConfig()

			out, _ := cmd.Flags().GetString("output")
			config.GetLogging().SetLogLevel("error")

			searchOpts := &wagon.StonesSearchOpts{
				Categories:       categories,
				Labels:           labels,
				LabelsMatches:    regLabels,
				Matches:          args,
				Hidden:           hidden,
				Names:            names,
				Annotations:      annotations,
				AndCondition:     !orCond,
				WithFiles:        files,
				WithRootfsPrefix: withRootfsPrefix,
				Full:             full,
			}

			var res *[]*wagon.Stone
			var resArts *[]*art.PackageArtifact
			var err error

			if len(packages) > 0 {
				for _, p := range packages {

					// NOTE: pass nil to ParsePackageStr because the
					//       search already is with Names options.
					pack, err := helpers.ParsePackageStr(nil, p)
					if err != nil {
						Fatal("Invalid package string ", p, ": ", err.Error())
					}
					searchOpts.Packages = append(searchOpts.Packages, pack)
				}

				if len(categories) == 0 && len(labels) == 0 &&
					len(regLabels) == 0 && len(args) == 0 {
					searchOpts.OnlyPackages = true
				}

			}

			maskManager := mask.NewPackagesMaskManager(config)
			if !ignoreMasks {
				err = maskManager.LoadFiles()
				if err != nil {
					Fatal("Error on load packages mask files.")
				}
			}

			searcher := wagon.NewSearcherSimple(config)
			defer searcher.Close()
			searcher.SetMaskManager(maskManager)

			if installed {
				res, err = searcher.SearchInstalled(searchOpts)
				if err != nil {
					fmt.Println("Error on retrieve installed packages ", err.Error())
					os.Exit(1)
				}
			} else {

				if artifactView {
					resArts, err = searcher.SearchArtifacts(searchOpts)
				} else {
					res, err = searcher.SearchStones(searchOpts)
				}
				if err != nil {
					fmt.Println("Error on retrieve installed packages ", err.Error())
					os.Exit(1)
				}
			}

			if out == "json" {
				var data []byte

				if artifactView {
					pack := art.ArtifactsPack{*resArts}
					data, err = json.Marshal(pack)
				} else {
					pack := wagon.StonesPack{*res}
					data, err = json.Marshal(pack)
				}
				if err != nil {
					fmt.Println("Error on marshal stones ", err.Error())
					os.Exit(1)
				}
				fmt.Println(string(data))
			} else if out == "yaml" {
				var data []byte

				if artifactView {
					pack := art.ArtifactsPack{*resArts}
					data, err = yaml.Marshal(pack)
				} else {
					pack := wagon.StonesPack{*res}
					data, err = yaml.Marshal(pack)
				}
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
	flags.StringSliceVarP(&names, "name", "n", []string{},
		"Search packages matching the package name string.")
	flags.Bool("condition-or", false,
		"The searching options are managed in OR between the searching types.")

	flags.StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	flags.Bool("hidden", false, "Include hidden packages")
	flags.Bool("files", false, "Show package files on YAML/JSON output.")
	flags.Bool("with-rootfs-prefix", true, "Add prefix of the configured rootfs path.")
	flags.Bool("table", false, "show output in a table (wider screens)")
	flags.Bool("quiet", false, "show output as list without version")
	flags.Bool("full", false, "Show full informations.")
	flags.Bool("ignore-masks", false, "Ignore packages masked.")
	flags.Bool("artifacts", false, "Show full artefact data.")

	return ans
}

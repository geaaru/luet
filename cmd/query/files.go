/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
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
			installed, _ := cmd.Flags().GetBool("installed")
			withRootfsPrefix, _ := cmd.Flags().GetBool("with-rootfs-prefix")

			searchOpts := &wagon.StonesSearchOpts{
				Categories:       []string{},
				Labels:           []string{},
				LabelsMatches:    []string{},
				Matches:          []string{},
				Hidden:           true,
				AndCondition:     false,
				WithFiles:        true,
				WithRootfsPrefix: withRootfsPrefix,
			}

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(config, a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				searchOpts.Packages = append(searchOpts.Packages, pack)
			}

			config.GetLogging().SetLogLevel("error")

			var res *[]*wagon.Stone
			var err error

			searcher := wagon.NewSearcherSimple(config)
			defer searcher.Close()

			if installed {
				res, err = searcher.SearchInstalled(searchOpts)
			} else {
				res, err = searcher.SearchStones(searchOpts)
			}
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

	flags := ans.Flags()

	flags.StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	flags.Bool("installed", false, "Search between system packages")
	flags.Bool("with-rootfs-prefix", true, "Add prefix of the configured rootfs path.")
	return ans
}

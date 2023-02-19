/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_query

import (
	"encoding/json"
	"fmt"

	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	solver "github.com/geaaru/luet/pkg/v2/solver"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewQueryOrphansCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "orphans [OPTIONS]",
		Short: "Show orphans packages.",
		Long: `An orphan package is a package that is no more
available in the configured and/or enabled repositories.

This operation could require a bit of time.
`,
		Aliases: []string{"o"},
		Run: func(cmd *cobra.Command, args []string) {
			out, _ := cmd.Flags().GetString("output")
			quiet, _ := cmd.Flags().GetBool("quiet")
			verbose, _ := cmd.Flags().GetBool("verbose")

			solveropts := &solver.SolverOpts{
				IgnoreConflicts: true,
				Force:           false,
				NoDeps:          false,
			}

			systemdb := config.GetSystemDB()
			s := solver.NewSolverImplementation("solverv2", config, solveropts)
			(*s).SetDatabase(systemdb)

			enableSpinner := false
			if out != "yaml" && out != "json" && verbose {
				enableSpinner = true

				InfoC(fmt.Sprintf(":rocket:%s %s",
					Bold(Blue("Luet")), Bold(Blue(util.Version()))))

				InfoC(":brain:Searching for orphans packages...")
				Spinner(3)
			}

			orphans, err := (*s).Orphans()
			if enableSpinner {
				SpinnerStop()
			}
			if err != nil {
				Fatal(err.Error())
			}

			switch out {
			case "json":
				list := pkg.NewPkgsList(orphans)
				data, err := json.Marshal(list)
				if err != nil {
					Fatal("Error on marshal packages", err.Error())
				}
				fmt.Println(string(data))
			case "yaml":
				list := pkg.NewPkgsList(orphans)
				data, err := yaml.Marshal(list)
				if err != nil {
					Fatal("Error on marshal packages", err.Error())
				}
				fmt.Println(string(data))
			default:
				for _, p := range *orphans {
					if quiet {
						fmt.Println(p.PackageName())
					} else {
						fmt.Println(p.HumanReadableString())
					}
				}
			}

		},
	}

	flags := ans.Flags()

	flags.Bool("verbose", true, "Show messages.")
	flags.Bool("quiet", false, "show output as list without version")
	flags.StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	return ans
}

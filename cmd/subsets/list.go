/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_subsets

import (
	"fmt"

	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"

	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

func NewSubsetsListCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "list [OPTIONS]",
		Short: "List of subsets enabled.",
		Args:  cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			quiet, _ := cmd.Flags().GetBool("quiet")

			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			if quiet {
				if len(config.Subsets.Enabled) > 0 {
					for _, s := range config.Subsets.Enabled {
						fmt.Println(s)
					}
				}
			} else {
				if len(config.Subsets.Enabled) == 0 {
					fmt.Println("No subsets enabled.")
				} else {

					InfoC(Bold(":ice_cream:Subsets enabled:"))
					for _, s := range config.Subsets.Enabled {
						InfoC(fmt.Sprintf(" * %s",
							Bold(Green(s))))
						sdef, ok := config.SubsetsDefinitions.Definitions[s]
						if ok {
							InfoC(Yellow(fmt.Sprintf(
								`   %s
    Num. Rules: %d`, sdef.Description, sdef.Rules)))
						} else if s == "portage" {
							InfoC(Yellow(
								`   Portage metadata and files.
`))
						} else if s == "devel" {
							InfoC(Yellow(
								`   Includes and devel files. Needed for compilation.
`))
						}
					}
				}

			}

		},
	}

	ans.Flags().BoolP("quiet", "q", false, "Show only name of the repositories.")

	return ans
}

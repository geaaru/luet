/*
Copyright © 2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_tree

import (
	"fmt"
	"os"

	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/v2/tree"

	"github.com/spf13/cobra"
)

func NewTreeGenIdx(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "genidx [OPTIONS]",
		Short: "Generate tree indexes.",
		Args:  cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			treePaths, _ := cmd.Flags().GetStringArray("tree")
			output, _ := cmd.Flags().GetString("output")

			opts := &tree.GenOpts{
				DryRun: dryRun,
			}

			for _, t := range treePaths {

				ti := tree.NewTreeIdx(t)
				err := ti.Generate(t, opts)
				if err != nil {
					fmt.Println("Error on generate indexes: " + err.Error())
					os.Exit(1)
				}

				switch output {
				case "yaml":
					data, err := ti.ToYAML()
					if err != nil {
						fmt.Println("Error on convert tree in YAML: " + err.Error())
						os.Exit(1)
					}

					fmt.Println(string(data))

				case "json":
					data, err := ti.ToJSON()
					if err != nil {
						fmt.Println("Error on convert tree in JSON: " + err.Error())
						os.Exit(1)
					}

					fmt.Println(string(data))
				}
			}

		},
	}

	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}

	flags := ans.Flags()
	flags.Bool("dry-run", false,
		"Generate indexes without update and/or create files.")
	flags.StringArrayP("tree", "t", []string{path},
		"Path of the tree to use.")
	ans.Flags().StringP("output", "o", "", "Output format ( Defaults: No output, available: json,yaml )")

	return ans
}

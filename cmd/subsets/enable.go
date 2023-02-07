/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_subsets

import (
	"fmt"
	"path/filepath"
	"strings"

	cfg "github.com/geaaru/luet/pkg/config"
	helpers "github.com/geaaru/luet/pkg/helpers/file"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"

	"github.com/spf13/cobra"
)

func NewSubsetsEnableCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "enable [OPTIONS] <subset1> ... <subsetN>",
		Short: "Enable one or more subsets.",
		Long: `Enable one or more subsets as subsets config file.

	$> luet subsets enable devel portage mysubset

	$> luet subsets enable -f my devel portage mysubset

The filename is used to write/update the file under the first
directory defined on subsets_confdir option (for example /etc/luet/subsets.conf.d/my.yml else main.yml is used).
`,
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			rootfs := ""
			subsetsEnabled := []string{}

			filename, _ := cmd.Flags().GetString("file")

			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			if len(config.SubsetsConfDir) == 0 {
				Fatal("No subsets config directories defined.")
			}

			// Respect the rootfs param on read repositories
			if !config.ConfigFromHost {
				rootfs, err = config.GetSystem().GetRootFsAbs()
				if err != nil {
					Fatal("Error on read rootfs config: ", err.Error())
				}
			}

			sconfdir := config.SubsetsConfDir[0]
			conffile := filepath.Join(rootfs, sconfdir, "main.yml")

			if filename != "" {
				conffile = filepath.Join(rootfs, sconfdir, filename+".yml")
			}

			var sc *cfg.LuetSubsetsConfig

			// Read the file if exists
			if helpers.Exists(conffile) {
				sc, err = cfg.NewSubsetsConfigFromFile(conffile)
				if err != nil {
					Fatal(err)
				}
			} else {
				sc = cfg.NewLuetSubsetsConfig()
			}

			for _, s := range args {
				if sc.HasSubset(s) {
					Warning(fmt.Sprintf(
						"subset %s already present. Skipped.", s))
					continue
				}

				subsetsEnabled = append(subsetsEnabled, s)
				sc.AddSubset(s)
			}

			err = sc.Write(conffile)
			if err != nil {
				Fatal(err)
			}

			if len(subsetsEnabled) > 0 {
				InfoC(fmt.Sprintf(
					"Subsets %s enabled :check_mark:.",
					strings.Join(subsetsEnabled, " ")))
			} else {
				InfoC("No subsets enabled.")
			}

		},
	}

	ans.Flags().StringP("file", "f", "",
		"Define the filename without extension where enable the subsets.")

	return ans
}

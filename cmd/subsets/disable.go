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

func NewSubsetsDisableCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "disable [OPTIONS] <subset1> ... <subsetN>",
		Short: "Disable one or more subsets.",
		Long: `Disable one or more subsets as subsets config file.

	$> luet subsets disable devel portage mysubset

	$> luet subsets disable -f my devel portage mysubset

The filename is used to write/update the file under the first
directory defined on subsets_confdir option (for example /etc/luet/subsets.conf.d/my.yml else main.yml is used).
`,
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			rootfs := ""
			subsetsDisabled := []string{}

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

			// Read the file if exists
			if !helpers.Exists(conffile) {
				Error(fmt.Sprintf(
					"The subsets config file %s doesn't exist.",
					filepath.Base(conffile)))
				Fatal("Maybe the subsets is enabled in another file or on luet.yaml?")
			}

			sc, err := cfg.NewSubsetsConfigFromFile(conffile)
			if err != nil {
				Fatal(err)
			}

			for _, s := range args {
				if !sc.HasSubset(s) {
					Warning(fmt.Sprintf(
						"subset %s not enabled. Skipped.", s))
					continue
				}

				subsetsDisabled = append(subsetsDisabled, s)
				sc.DelSubset(s)
			}

			err = sc.Write(conffile)
			if err != nil {
				Fatal(err)
			}

			if len(subsetsDisabled) > 0 {
				InfoC(fmt.Sprintf(
					"Subsets %s disabled :check_mark:.",
					strings.Join(subsetsDisabled, " ")))
			} else {
				InfoC("No subsets to disable.")
			}

		},
	}

	ans.Flags().StringP("file", "f", "",
		"Define the filename without extension where enable the subsets.")

	return ans
}

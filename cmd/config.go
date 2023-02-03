/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"

	config "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func newConfigCommand(cfg *config.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:     "config",
		Short:   "Print config",
		Long:    `Show luet configuration`,
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			// Load config protect configs
			installer.LoadConfigProtectConfs(cfg)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(cfg)
			// Load subsets config
			subsets.LoadSubsetsConfig(cfg)

			data, err := cfg.YAML()
			if err != nil {
				Fatal(err.Error())
			}

			fmt.Println(string(data))
		},
	}

	return ans
}

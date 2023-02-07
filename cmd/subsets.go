/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	cmd_subsets "github.com/geaaru/luet/cmd/subsets"
	cfg "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

func newSubsetsCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "subsets [command] [OPTIONS]",
		Short: "Manage subsets",
	}

	ans.AddCommand(
		cmd_subsets.NewSubsetsListCommand(config),
		cmd_subsets.NewSubsetsEnableCommand(config),
		cmd_subsets.NewSubsetsDisableCommand(config),
	)

	return ans
}

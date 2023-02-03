/*
Copyright © 2021-2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/geaaru/luet/cmd/repo"
	cfg "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

func newRepoCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "repo [command] [OPTIONS]",
		Short: "Manage repositories",
	}

	ans.AddCommand(
		NewRepoListCommand(config),
		NewRepoUpdateCommand(config),
		NewRepoEnableCommand(config),
		NewRepoDisableCommand(config),
	)

	return ans
}

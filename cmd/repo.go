/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/macaroni-os/anise/cmd/repo"
	cfg "github.com/macaroni-os/anise/pkg/config"

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

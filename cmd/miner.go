/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/geaaru/luet/cmd/miner"
	cfg "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

func newMinerCommand(config *cfg.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:    "miner [command] [OPTIONS]",
		Hidden: true,
		Short:  "Advance Users/Develpers only commands.",
	}

	ans.AddCommand(
		NewDownload(config),
		NewInstallPackage(config),
		NewRemovePackage(config),
		NewReinstallPackage(config),
		NewReplacePackage(config),
	)

	return ans
}

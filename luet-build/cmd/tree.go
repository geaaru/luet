/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	. "github.com/geaaru/luet/luet-build/cmd/tree"
	cfg "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

func newTreeCommand(config *cfg.LuetConfig) *cobra.Command {

	var treeGroupCmd = &cobra.Command{
		Use:   "tree [command] [OPTIONS]",
		Short: "Tree operations",
	}

	treeGroupCmd.AddCommand(
		NewTreePkglistCommand(config),
		NewTreeValidateCommand(),
		NewTreeBumpCommand(),
		NewTreeImageCommand(),
	)

	return treeGroupCmd
}

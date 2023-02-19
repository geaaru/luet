/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd

import (
	. "github.com/geaaru/luet/cmd/query"
	cfg "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

func newQueryCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "query [command] [OPTIONS]",
		Short:   "Repository query tools.",
		Aliases: []string{"q"},
	}

	ans.AddCommand(
		NewQueryFilesCommand(config),
		NewQueryBelongsCommand(config),
		NewQueryOrphansCommand(config),
	)

	return ans
}

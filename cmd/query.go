/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd

import (
	. "github.com/macaroni-os/anise/cmd/query"
	cfg "github.com/macaroni-os/anise/pkg/config"

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

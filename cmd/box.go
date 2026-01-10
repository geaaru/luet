/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd

import (
	. "github.com/macaroni-os/anise/cmd/box"
	config "github.com/macaroni-os/anise/pkg/config"

	"github.com/spf13/cobra"
)

var boxGroupCmd = &cobra.Command{
	Use:   "box [command] [OPTIONS]",
	Short: "Manage anise boxes",
}

func newBoxCommand(cfg *config.AniseConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "box [command] [OPTIONS]",
		Short: "Manage Box/Sandbox",
	}

	ans.AddCommand(
		NewBoxExecCommand(cfg),
	)

	return ans
}

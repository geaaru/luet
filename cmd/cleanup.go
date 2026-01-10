/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"github.com/macaroni-os/anise/cmd/util"
	cfg "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"
	installer "github.com/macaroni-os/anise/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func newCleanupCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "cleanup",
		Short: "Clean packages cache.",
		Long:  `remove downloaded packages tarballs and clean cache directory`,
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSystemFlags(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.SetSystemConfig()

			purge, _ := cmd.Flags().GetBool("purge-repos")

			aManager := installer.NewArtifactsManager(config)
			defer aManager.Close()

			err := aManager.CleanLocalPackagesCache()
			if err != nil {
				Fatal(err.Error())
			}

			if purge {
				err = aManager.PurgeLocalReposCache()
				if err != nil {
					Fatal(err.Error())
				}
			}
		},
	}

	ans.Flags().String("system-dbpath", "", "System db path")
	ans.Flags().String("system-target", "", "System rootpath")
	ans.Flags().String("system-engine", "", "System DB engine")
	ans.Flags().Bool("purge-repos", false,
		"Remove all repos files. This impacts on searching packages too.")

	return ans
}

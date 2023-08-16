// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
//	Daniele Rondina <geaaru@sabayonlinux.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.
package cmd

import (
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	installer "github.com/geaaru/luet/pkg/v2/installer"

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

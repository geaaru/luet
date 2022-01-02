// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//                  Daniele Rondina <geaaru@sabayonlinux.org>
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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mudler/luet/cmd/util"
	cfg "github.com/mudler/luet/pkg/config"
	fileHelper "github.com/mudler/luet/pkg/helpers/file"
	. "github.com/mudler/luet/pkg/logger"

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
			var cleaned int = 0
			util.SetSystemConfig()

			purge, _ := cmd.Flags().GetBool("purge-repos")

			// Check if cache dir exists
			if fileHelper.Exists(config.GetSystem().GetSystemPkgsCacheDirPath()) {

				files, err := ioutil.ReadDir(config.GetSystem().GetSystemPkgsCacheDirPath())
				if err != nil {
					Fatal("Error on read cachedir ", err.Error())
				}

				for _, file := range files {
					if file.IsDir() {
						continue
					}

					if config.GetGeneral().Debug {
						Info("Removing ", file.Name())
					}

					err := os.RemoveAll(
						filepath.Join(config.GetSystem().GetSystemPkgsCacheDirPath(), file.Name()))
					if err != nil {
						Fatal("Error on removing", file.Name())
					}
					cleaned++
				}
			}

			Info("Cleaned: ", cleaned, "packages.")

			if purge {

				reposDir := config.GetSystem().GetSystemReposDirPath()
				cnt := 0

				Debug("Repositories dir:", reposDir)

				if fileHelper.Exists(reposDir) {

					files, err := ioutil.ReadDir(reposDir)
					if err != nil {
						Fatal("Error on read reposdir", err.Error())
					}

					for _, file := range files {
						if !file.IsDir() {
							continue
						}

						d := filepath.Join(reposDir, file.Name())

						err := os.RemoveAll(d)
						if err != nil {
							Fatal("Error on removing dir", d)
						}

						cnt++
					}

					Info("Repos Cleaned: ", cnt)

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

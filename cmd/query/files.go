// Copyright © 2021 Daniele Rondina <geaaru@funtoo.org>
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

package cmd_query

import (
	"encoding/json"
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewQueryFilesCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:     "files <pkg1> ... <pkgN> [OPTIONS]",
		Short:   "Show files owned by a specific package.",
		Aliases: []string{"fi", "f"},
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Missing package")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			var pkgs pkg.Packages

			out, _ := cmd.Flags().GetString("output")

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				pkgs = append(pkgs, pack)
			}

			util.SetSystemConfig()
			util.SetSolverConfig()

			config.GetLogging().SetLogLevel("error")
			Debug("Solver", config.GetSolverOptions().CompactString())
			repos := installer.SystemRepositories(config)

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
				Concurrency:                 config.GetGeneral().Concurrency,
				SolverOptions:               *config.GetSolverOptions(),
				PreserveSystemEssentialData: true,
				SyncRepositories:            false,
			})
			inst.Repositories(repos)

			synced, err := inst.GetRepositoriesInstances(true)
			if err != nil {
				Fatal("Error: " + err.Error())
			}

			pkgs = synced.ResolveSelectors(pkgs)
			matches := synced.PackageMatches(pkgs)

			if len(matches) > 0 {

				ans := []string{}
				var data []byte

				for _, m := range matches {

					files := m.Artifact.Files

					for _, f := range files {

						switch out {
						case "yaml", "json":
							ans = append(ans, f)
						default:
							fmt.Println(f)
						}
					}

				}

				if out == "yaml" || out == "json" {
					switch out {
					case "yaml":
						data, err = yaml.Marshal(ans)
					case "json":
						data, err = json.Marshal(ans)
					}

					if err != nil {
						Fatal("Error on marshal data:", err.Error())
					}

					fmt.Println(string(data))
				}
			}

		},
	}

	ans.Flags().StringP("output", "o", "terminal",
		"Output format ( Defaults: terminal, available: json,yaml )")
	return ans
}

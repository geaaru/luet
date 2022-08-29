// Copyright © 2019-2021 Ettore Di Giacinto <mudler@gentoo.org>
//                       Daniele Rondina <geaaru@sabayonlinux.org>
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
	"fmt"
	"os"

	helpers "github.com/geaaru/luet/cmd/helpers"
	cmdrepo "github.com/geaaru/luet/cmd/repo"
	"github.com/geaaru/luet/cmd/util"
	cfg "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
	"github.com/geaaru/luet/pkg/subsets"

	"github.com/spf13/cobra"
)

func newInstallCommand(config *cfg.LuetConfig) *cobra.Command {

	var ans = &cobra.Command{
		Use:   "install <pkg1> <pkg2> ...",
		Short: "Install a package",
		Long: `Installs one or more packages without asking questions:

	$ luet install -y utils/busybox utils/yq ...
	
To install only deps of a package:
	
	$ luet install --onlydeps utils/busybox ...
	
To not install deps of a package:
	
	$ luet install --nodeps utils/busybox ...

To force install a package:
	
	$ luet install --force utils/busybox ...
`,
		Aliases: []string{"i"},
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSolverFlags(cmd)
			config.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			config.Viper.BindPFlag("nodeps", cmd.Flags().Lookup("nodeps"))
			config.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			config.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
			config.Viper.BindPFlag("general.overwrite_dir_perms",
				cmd.Flags().Lookup("Overwrite exiting directories permissions."))
		},
		Run: func(cmd *cobra.Command, args []string) {
			var toInstall pkg.Packages

			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toInstall = append(toInstall, pack)
			}

			force := config.Viper.GetBool("force")
			nodeps := config.Viper.GetBool("nodeps")
			onlydeps := config.Viper.GetBool("onlydeps")
			yes := config.Viper.GetBool("yes")
			downloadOnly, _ := cmd.Flags().GetBool("download-only")
			finalizerEnvs, _ := cmd.Flags().GetStringArray("finalizer-env")
			relax, _ := cmd.Flags().GetBool("relax")
			skipFinalizers, _ := cmd.Flags().GetBool("skip-finalizers")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")

			if syncRepos {

				var ch chan util.ChannelRepoOpRes = make(
					chan util.ChannelRepoOpRes,
					config.GetGeneral().Concurrency,
				)
				// Using new way
				nOps := 0

				for idx, repo := range config.SystemRepositories {
					if repo.Enable {
						go cmdrepo.ProcessRepository(&config.SystemRepositories[idx], config, ch, force)
						nOps++
					}
				}

				res := 0
				if nOps > 0 {
					for i := 0; i < nOps; i++ {
						resp := <-ch
						if resp.Error != nil && !force {
							res = 1
							Error("Error on update repository " + resp.Repo.Name + ": " + resp.Error.Error())
						}
					}
				} else {
					fmt.Println("No repositories candidates found.")
				}

				if res != 0 {
					os.Exit(res)
				}

			}

			util.SetSolverConfig()

			Debug("Solver", config.GetSolverOptions().CompactString())
			repos := installer.SystemRepositories(config)

			// Load config protect configs
			installer.LoadConfigProtectConfs(config)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(config)
			// Load subsets config
			subsets.LoadSubsetsConfig(config)

			// Load finalizer runtime environments
			err := util.SetCliFinalizerEnvs(finalizerEnvs)
			if err != nil {
				Fatal(err.Error())
			}

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
				Concurrency:                 config.GetGeneral().Concurrency,
				SolverOptions:               *config.GetSolverOptions(),
				NoDeps:                      nodeps,
				Force:                       force,
				OnlyDeps:                    onlydeps,
				PreserveSystemEssentialData: true,
				DownloadOnly:                downloadOnly,
				Ask:                         !yes,
				Relaxed:                     relax,
				SkipFinalizers:              skipFinalizers,
				SyncRepositories:            false,
			})
			inst.Repositories(repos)

			system := &installer.System{
				Database: config.GetSystemDB(),
				Target:   config.GetSystem().Rootfs,
			}
			err = inst.Install(toInstall, system)
			if err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	ans.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	ans.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	ans.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	ans.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	ans.Flags().Bool("nodeps", false, "Don't consider package dependencies (harmful!)")
	ans.Flags().Bool("relax", false, "Relax installation constraints")

	ans.Flags().Bool("onlydeps", false, "Consider **only** package dependencies")
	ans.Flags().Bool("force", false, "Skip errors and keep going (potentially harmful)")
	ans.Flags().Bool("solver-concurrent", false, "Use concurrent solver (experimental)")
	ans.Flags().BoolP("yes", "y", false, "Don't ask questions")
	ans.Flags().Bool("download-only", false, "Download only")
	ans.Flags().StringArray("finalizer-env", []string{},
		"Set finalizer environment in the format key=value.")
	ans.Flags().Bool("overwrite-existing-dir-perms", false,
		"Overwrite exiting directories permissions.")
	ans.Flags().Bool("skip-finalizers", false,
		"Skip the execution of the finalizers.")
	ans.Flags().Bool("sync-repos", false,
		"Sync repositories before install. Note: If there are in memory repositories then the sync is done always.")

	return ans
}

// Copyright © 2020-2021 Ettore Di Giacinto <mudler@mocaccino.org>
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
	config "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	"github.com/geaaru/luet/pkg/solver"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"

	"github.com/spf13/cobra"
)

func newReplaceCommand(cfg *config.LuetConfig) *cobra.Command {

	var replaceCmd = &cobra.Command{
		Use:     "replace <pkg1> <pkg2> --for <pkg3> --for <pkg4> ...",
		Short:   "replace a set of packages",
		Aliases: []string{"r"},
		Long: `Replaces one or a group of packages without asking questions:

		$ luet replace -y system/busybox ... --for shells/bash --for system/coreutils ...
	`,
		PreRun: func(cmd *cobra.Command, args []string) {
			util.BindSolverFlags(cmd)
			cfg.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			cfg.Viper.BindPFlag("nodeps", cmd.Flags().Lookup("nodeps"))
			cfg.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
			cfg.Viper.BindPFlag("for", cmd.Flags().Lookup("for"))

			cfg.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			var toUninstall pkg.Packages
			var toAdd pkg.Packages

			f := cfg.Viper.GetStringSlice("for")
			force := cfg.Viper.GetBool("force")
			nodeps := cfg.Viper.GetBool("nodeps")
			onlydeps := cfg.Viper.GetBool("onlydeps")
			yes := cfg.Viper.GetBool("yes")
			downloadOnly, _ := cmd.Flags().GetBool("download-only")
			syncRepos, _ := cmd.Flags().GetBool("sync-repos")

			util.SetSolverConfig()
			for _, a := range args {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toUninstall = append(toUninstall, pack)
			}

			for _, a := range f {
				pack, err := helpers.ParsePackageStr(a)
				if err != nil {
					Fatal("Invalid package string ", a, ": ", err.Error())
				}
				toAdd = append(toAdd, pack)
			}

			// This shouldn't be necessary, but we need to unmarshal the repositories to a concrete struct, thus we need to port them back to the Repositories type
			repos := installer.Repositories{}
			for _, repo := range cfg.SystemRepositories {
				if !repo.Enable {
					continue
				}
				r := installer.NewSystemRepository(repo)
				repos = append(repos, r)
			}

			Debug("Solver", cfg.GetSolverOptions().CompactString())

			// Load config protect configs
			installer.LoadConfigProtectConfs(cfg)

			inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
				Concurrency:                 cfg.GetGeneral().Concurrency,
				SolverOptions:               *cfg.GetSolverOptions(),
				NoDeps:                      nodeps,
				Force:                       force,
				OnlyDeps:                    onlydeps,
				PreserveSystemEssentialData: true,
				Ask:                         !yes,
				DownloadOnly:                downloadOnly,
				SyncRepositories:            syncRepos,
			})
			inst.Repositories(repos)

			system := &installer.System{
				Database: cfg.GetSystemDB(),
				Target:   cfg.GetSystem().Rootfs,
			}
			err := inst.Swap(toUninstall, toAdd, system)
			if err != nil {
				Fatal("Error: " + err.Error())
			}
		},
	}

	replaceCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	replaceCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	replaceCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	replaceCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	replaceCmd.Flags().Bool("nodeps", false, "Don't consider package dependencies (harmful!)")
	replaceCmd.Flags().Bool("onlydeps", false, "Consider **only** package dependencies")
	replaceCmd.Flags().Bool("force", false, "Skip errors and keep going (potentially harmful)")
	replaceCmd.Flags().Bool("solver-concurrent", false, "Use concurrent solver (experimental)")
	replaceCmd.Flags().BoolP("yes", "y", false, "Don't ask questions")
	replaceCmd.Flags().StringSlice("for", []string{}, "Packages that has to be installed in place of others")
	replaceCmd.Flags().Bool("download-only", false, "Download only")
	replaceCmd.Flags().Bool("sync-repos", false,
		"Sync repositories before replace. Note: If there are in memory repositories then the sync is done always.")
	return replaceCmd
}

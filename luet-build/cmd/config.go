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
	"fmt"

	"github.com/geaaru/luet/luet-build/pkg/installer"
	config "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/subsets"

	"github.com/spf13/cobra"
)

func newConfigCommand(cfg *config.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:     "config",
		Short:   "Print config",
		Long:    `Show luet configuration`,
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			//util.SetSolverConfig()

			// Load config protect configs
			installer.LoadConfigProtectConfs(cfg)
			// Load subsets defintions
			subsets.LoadSubsetsDefintions(cfg)
			// Load subsets config
			subsets.LoadSubsetsConfig(cfg)

			data, err := cfg.YAML()
			if err != nil {
				Fatal(err.Error())
			}

			fmt.Println(string(data))
		},
	}

	return ans
}

// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
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

package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/config"

	"github.com/spf13/cobra"
)

type ChannelRepoOpRes struct {
	Error error
	Repo  *config.LuetRepository
}

func Version() string {
	if config.BuildGoVersion != "" {
		return fmt.Sprintf("%s-%s-g%s %s - %s",
			config.LuetVersion, config.LuetForkVersion, config.BuildCommit,
			config.BuildTime, config.BuildGoVersion)
	} else {
		return fmt.Sprintf("%s-%s-g%s %s", config.LuetVersion,
			config.LuetForkVersion, config.BuildCommit, config.BuildTime)
	}
}

func BindSystemFlags(cmd *cobra.Command) {
	LuetCfg.Viper.BindPFlag("system.database_path", cmd.Flags().Lookup("system-dbpath"))
	LuetCfg.Viper.BindPFlag("system.rootfs", cmd.Flags().Lookup("system-target"))
	LuetCfg.Viper.BindPFlag("system.database_engine", cmd.Flags().Lookup("system-engine"))
}

func BindSolverFlags(cmd *cobra.Command) {
	LuetCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
	LuetCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
	LuetCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
	LuetCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
	LuetCfg.Viper.BindPFlag("solver.implementation", cmd.Flags().Lookup("solver-implementation"))
}

func SetSystemConfig() {
	dbpath := LuetCfg.Viper.GetString("system.database_path")
	rootfs := LuetCfg.Viper.GetString("system.rootfs")
	engine := LuetCfg.Viper.GetString("system.database_engine")

	LuetCfg.System.DatabaseEngine = engine
	LuetCfg.System.DatabasePath = dbpath
	LuetCfg.System.SetRootFS(rootfs)
}

func SetCliFinalizerEnvs(finalizerEnvs []string) error {
	if len(finalizerEnvs) > 0 {
		for _, v := range finalizerEnvs {
			idx := strings.Index(v, "=")
			if idx < 0 {
				return errors.New("Found invalid runtime finalizer environment: " + v)
			}

			LuetCfg.SetFinalizerEnv(v[0:idx], v[idx+1:])
		}

	}

	return nil
}

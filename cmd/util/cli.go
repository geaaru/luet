/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/config"

	"github.com/spf13/cobra"
)

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

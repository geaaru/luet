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
			config.AniseVersion, config.AniseForkVersion, config.BuildCommit,
			config.BuildTime, config.BuildGoVersion)
	} else {
		return fmt.Sprintf("%s-%s-g%s %s", config.AniseVersion,
			config.AniseForkVersion, config.BuildCommit, config.BuildTime)
	}
}

func BindSystemFlags(cmd *cobra.Command) {
	AniseCfg.Viper.BindPFlag("system.database_path", cmd.Flags().Lookup("system-dbpath"))
	AniseCfg.Viper.BindPFlag("system.rootfs", cmd.Flags().Lookup("system-target"))
	AniseCfg.Viper.BindPFlag("system.database_engine", cmd.Flags().Lookup("system-engine"))
}

func BindSolverFlags(cmd *cobra.Command) {
	AniseCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
	AniseCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
	AniseCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
	AniseCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
	AniseCfg.Viper.BindPFlag("solver.implementation", cmd.Flags().Lookup("solver-implementation"))
}

func SetSystemConfig() {
	dbpath := AniseCfg.Viper.GetString("system.database_path")
	rootfs := AniseCfg.Viper.GetString("system.rootfs")
	engine := AniseCfg.Viper.GetString("system.database_engine")

	AniseCfg.System.DatabaseEngine = engine
	AniseCfg.System.DatabasePath = dbpath
	AniseCfg.System.SetRootFS(rootfs)
}

func SetCliFinalizerEnvs(finalizerEnvs []string) error {
	if len(finalizerEnvs) > 0 {
		for _, v := range finalizerEnvs {
			idx := strings.Index(v, "=")
			if idx < 0 {
				return errors.New("Found invalid runtime finalizer environment: " + v)
			}

			AniseCfg.SetFinalizerEnv(v[0:idx], v[idx+1:])
		}

	}

	return nil
}

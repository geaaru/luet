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
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
	"github.com/marcsauter/single"

	config "github.com/geaaru/luet/pkg/config"
	helpers "github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	repo "github.com/geaaru/luet/pkg/repository"
	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
	"github.com/spf13/cobra"
)

var cfgFile string
var Verbose bool
var LockedCommands = []string{"install", "uninstall", "upgrade"}

func version() string {
	if config.BuildGoVersion != "" {
		return fmt.Sprintf("%s-%s-g%s %s - %s",
			config.LuetVersion, config.LuetForkVersion, config.BuildCommit,
			config.BuildTime, config.BuildGoVersion)
	} else {
		return fmt.Sprintf("%s-%s-g%s %s", config.LuetVersion,
			config.LuetForkVersion, config.BuildCommit, config.BuildTime)
	}
}

var bannerCommands = []string{
	"uninstall", "install", "build",
}

func displayVersionBanner() {
	display := false
	if len(os.Args) > 1 {
		for _, c := range bannerCommands {
			if os.Args[1] == c {
				display = true
			}
		}
	}
	if display {
		Info("Luet version", version())
	}
}

func handleLock() {
	if os.Getenv("LUET_NOLOCK") != "true" {
		if len(os.Args) > 1 {
			for _, lockedCmd := range LockedCommands {
				if os.Args[1] == lockedCmd {
					s := single.New("luet")
					if err := s.CheckLock(); err != nil && err == single.ErrAlreadyRunning {
						Fatal("another instance of the app is already running, exiting")
					} else if err != nil {
						// Another error occurred, might be worth handling it as well
						Fatal("failed to acquire exclusive app lock:", err.Error())
					}
					defer s.TryUnlock()
					break
				}
			}
		}
	}
}

func LoadConfig(c *config.LuetConfig) error {
	// If a config file is found, read it in.
	err := c.Viper.ReadInConfig()
	if err != nil {
		Debug(fmt.Sprintf("Error on reading file %s: %s",
			c.Viper.ConfigFileUsed(), err.Error()))
	}

	err = c.Viper.Unmarshal(&c)
	if err != nil {
		return err
	}

	noSpinner := c.Viper.GetBool("no_spinner")

	InitAurora()
	if !noSpinner {
		NewSpinner()
	}

	Debug("Using config file:", c.Viper.ConfigFileUsed())

	if c.GetLogging().EnableLogFile && c.GetLogging().Path != "" {
		// Init zap logger
		err = ZapLogger()
		if err != nil {
			return err
		}
	}

	// Load repositories
	err = repo.LoadRepositories(c)
	if err != nil {
		return err
	}

	// Initialize default tarformers instance
	// to use the config object used by the library.
	cfg := tarf_specs.NewConfig(c.Viper)
	if c.GetLogging().Paranoid {
		cfg.GetGeneral().Debug = true
		cfg.GetLogging().Level = c.GetLogging().Level
	} else {
		cfg.GetGeneral().Debug = false
		cfg.GetLogging().Level = "warning"
	}

	t := tarf.NewTarFormersWithLog(cfg, true)
	tarf.SetDefaultTarFormers(t)

	return nil
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var cfg *config.LuetConfig = config.NewLuetConfig(nil)
	config.LuetCfg = cfg

	initConfig(cfg)

	// RootCmd represents the base command when called without any subcommands
	var RootCmd = &cobra.Command{
		Use:   "luet",
		Short: "Container based package manager",
		Long: `Luet is a single-binary package manager based on containers to build packages.
		
	To install a package:

		$ luet install package

	To search for a package in the repositories:

	$ luet search package

	To list all packages installed in the system:

		$ luet search --installed .

	To show hidden packages:

		$ luet search --hidden package
		
	`,
		Version: version(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

			cfg.Viper.SetConfigType("yaml")

			if cfgFile != "" { // enable ability to specify config file via flag
				cfg.Viper.SetConfigFile(cfgFile)
			} else {
				// Retrieve pwd directory
				pwdDir, err := os.Getwd()
				if err != nil {
					Error(err)
					os.Exit(1)
				}
				homeDir := helpers.GetHomeDir()

				if fileHelper.Exists(filepath.Join(pwdDir, ".luet.yaml")) || (homeDir != "" && fileHelper.Exists(filepath.Join(homeDir, ".luet.yaml"))) {
					cfg.Viper.AddConfigPath(".")
					if homeDir != "" {
						cfg.Viper.AddConfigPath(homeDir)
					}
					cfg.Viper.SetConfigName(".luet")
				} else {
					cfg.Viper.SetConfigName("luet")
					cfg.Viper.AddConfigPath("/etc/luet")
				}
			}

			err := LoadConfig(cfg)
			if err != nil {
				Fatal("failed to load configuration:", err.Error())
			}
			// Initialize tmpdir prefix. TODO: Move this with LoadConfig
			// directly on sub command to ensure the creation only when it's
			// needed.
			err = cfg.GetSystem().InitTmpDir()
			if err != nil {
				Fatal("failed on init tmp basedir:", err.Error())
			}

			handleLock()
			displayVersionBanner()

		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Cleanup all tmp directories used by luet
			err := cfg.GetSystem().CleanupTmpDir()
			if err != nil {
				Warning("failed on cleanup tmpdir:", err.Error())
			}

			systemDB := cfg.GetSystemDB()
			err = systemDB.Close()
			if err != nil {
				Warning("failed on close database:", err.Error())
			}

		},
		SilenceErrors: true,
	}

	initCommand(RootCmd, cfg)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func initCommand(rootCmd *cobra.Command, cfg *config.LuetConfig) {
	pflags := rootCmd.PersistentFlags()
	pflags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.luet.yaml)")
	pflags.BoolP("debug", "d", false, "verbose output")
	pflags.Bool("fatal", false, "Enables Warnings to exit")
	pflags.Bool("enable-logfile", false, "Enable log to file")
	pflags.Bool("no-spinner", false, "Disable spinner.")
	pflags.Bool("color", cfg.Viper.GetBool("logging.color"), "Enable/Disable color.")
	pflags.Bool("emoji", cfg.Viper.GetBool("logging.enable_emoji"), "Enable/Disable emoji.")
	pflags.Bool("skip-config-protect", cfg.Viper.GetBool("config_protect_skip"),
		"Disable config protect analysis.")
	pflags.StringP("logfile", "l", cfg.Viper.GetString("logging.path"),
		"Logfile path. Empty value disable log to file.")
	pflags.String("system-dbpath", "", "System db path")
	pflags.String("system-target", "", "System rootpath")
	pflags.String("system-engine", "", "System DB engine")

	// os/user doesn't work in from scratch environments.
	// Check if i can retrieve user informations.
	_, err := user.Current()
	if err != nil {
		Warning("failed to retrieve user identity:", err.Error())
	}
	pflags.Bool("same-owner", cfg.Viper.GetBool("general.same_owner"),
		"Maintain same owner on uncompress.")
	pflags.Int("concurrency", runtime.NumCPU(), "Concurrency")

	cfg.Viper.BindPFlag("logging.color", pflags.Lookup("color"))
	cfg.Viper.BindPFlag("logging.enable_emoji", pflags.Lookup("emoji"))
	cfg.Viper.BindPFlag("logging.enable_logfile", pflags.Lookup("enable-logfile"))
	cfg.Viper.BindPFlag("logging.path", pflags.Lookup("logfile"))

	cfg.Viper.BindPFlag("general.concurrency", pflags.Lookup("concurrency"))
	cfg.Viper.BindPFlag("general.debug", pflags.Lookup("debug"))
	cfg.Viper.BindPFlag("general.fatal_warnings", pflags.Lookup("fatal"))
	cfg.Viper.BindPFlag("general.same_owner", pflags.Lookup("same-owner"))

	// Currently I maintain this only from cli.
	cfg.Viper.BindPFlag("no_spinner", pflags.Lookup("no-spinner"))
	cfg.Viper.BindPFlag("config_protect_skip", pflags.Lookup("skip-config-protect"))

	cfg.Viper.BindPFlag("system.database_path", pflags.Lookup("system-dbpath"))
	cfg.Viper.BindPFlag("system.rootfs", pflags.Lookup("system-target"))
	cfg.Viper.BindPFlag("system.database_engine", pflags.Lookup("system-engine"))

	// Add main commands
	rootCmd.AddCommand(
		newBoxCommand(cfg),
		newConfigCommand(cfg),
		newDatabaseCommand(cfg),
		newExecCommand(cfg),
		newRepoCommand(cfg),
		newReinstallCommand(cfg),
		newReclaimCommand(cfg),
		newReplaceCommand(cfg),
		newUpgradeCommand(cfg),
		newSearchCommand(cfg),
		newCleanupCommand(cfg),
		newQueryCommand(cfg),
		newMinerCommand(cfg),
		newUninstallCommand(cfg),
		newInstallCommand(cfg),
	)

}

// initConfig reads in config file and ENV variables if set.
func initConfig(cfg *config.LuetConfig) {
	// Luet support these priorities on read configuration file:
	// - command line option (if available)
	// - $PWD/.luet.yaml
	// - $HOME/.luet.yaml
	// - /etc/luet/luet.yaml
	//
	// Note: currently a single viper instance support only one config name.

	cfg.Viper.SetEnvPrefix(config.LuetEnvPrefix)

	cfg.Viper.BindEnv("config")
	cfg.Viper.SetDefault("config", "")
	cfg.Viper.SetDefault("etcd-config", false)

	cfg.Viper.AutomaticEnv() // read in environment variables that match

	// Create EnvKey Replacer for handle complex structure
	replacer := strings.NewReplacer(".", "__")
	cfg.Viper.SetEnvKeyReplacer(replacer)

	cfg.Viper.SetTypeByDefaultValue(true)
}

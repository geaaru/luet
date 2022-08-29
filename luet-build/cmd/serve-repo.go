/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"net/http"
	"os"

	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	"github.com/spf13/cobra"
)

func newServerRepoCommand(config *cfg.LuetConfig) *cobra.Command {

	var serverepoCmd = &cobra.Command{
		Use:   "serve-repo",
		Short: "Embedded micro-http server",
		Long:  `Embedded mini http server for serving local repositories`,
		PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("dir", cmd.Flags().Lookup("dir"))
			config.Viper.BindPFlag("address", cmd.Flags().Lookup("address"))
			config.Viper.BindPFlag("port", cmd.Flags().Lookup("port"))
		},
		Run: func(cmd *cobra.Command, args []string) {

			dir := config.Viper.GetString("dir")
			port := config.Viper.GetString("port")
			address := config.Viper.GetString("address")

			http.Handle("/", http.FileServer(http.Dir(dir)))

			Info("Serving ", dir, " on HTTP port: ", port)
			Fatal(http.ListenAndServe(address+":"+port, nil))
		},
	}

	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	serverepoCmd.Flags().String("dir", path, "Packages folder (output from build)")
	serverepoCmd.Flags().String("port", "9090", "Listening port")
	serverepoCmd.Flags().String("address", "0.0.0.0", "Listening address")

	return serverepoCmd
}

// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
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
	"os"
	"path/filepath"
	"time"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/pkg/compiler/types/artifact"
	"github.com/geaaru/luet/pkg/compiler/types/compression"
	compilerspec "github.com/geaaru/luet/pkg/compiler/types/spec"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	"github.com/spf13/cobra"
)

func newPackCommand(config *cfg.LuetConfig) *cobra.Command {

	var packCmd = &cobra.Command{
		Use:   "pack <package name>",
		Short: "pack a custom package",
		Long: `Pack creates a package from a directory, generating the metadata required from a tree to generate a repository.

	Pack can be used to manually replace what "luet build" does automatically by reading the packages build.yaml files.

		$ mkdir -p output/etc/foo
		$ echo "my config" > output/etc/foo
		$ luet pack foo/bar@1.1 --source output

	Afterwards, you can use the content generated and associate it with a tree and a corresponding definition.yaml file with "luet create-repo".
	`,

		PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("destination", cmd.Flags().Lookup("destination"))
			config.Viper.BindPFlag("compression", cmd.Flags().Lookup("compression"))
			config.Viper.BindPFlag("source", cmd.Flags().Lookup("source"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			sourcePath := config.Viper.GetString("source")

			dst := config.Viper.GetString("destination")
			compressionType := config.Viper.GetString("compression")
			concurrency := config.GetGeneral().Concurrency

			if len(args) != 1 {
				Fatal("You must specify a package name")
			}

			packageName := args[0]

			p, err := helpers.ParsePackageStr(config, packageName)
			if err != nil {
				Fatal("Invalid package string ", packageName, ": ", err.Error())
			}

			spec := &compilerspec.LuetCompilationSpec{Package: p}
			a := artifact.NewPackageArtifact(filepath.Join(dst, p.GetFingerPrint()+".package.tar"))
			a.CompressionType = compression.Implementation(compressionType)
			err = a.Compress(sourcePath, concurrency)
			if err != nil {
				Fatal("failed compressing ", packageName, ": ", err.Error())
			}
			a.CompileSpec = spec
			filelist, err := a.FileList()
			if err != nil {
				Fatal("failed generating file list for ", packageName, ": ", err.Error())
			}
			a.Files = filelist
			a.CompileSpec.GetPackage().SetBuildTimestamp(time.Now().String())
			err = a.WriteYaml(dst)
			if err != nil {
				Fatal("failed writing metadata yaml file for ", packageName, ": ", err.Error())
			}
		},
	}

	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	packCmd.Flags().String("source", path, "Source folder")
	packCmd.Flags().String("destination", path, "Destination folder")
	packCmd.Flags().String("compression", "gzip", "Compression alg: none, gzip")

	return packCmd
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_database

import (
	"io/ioutil"

	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	installer "github.com/geaaru/luet/pkg/v2/installer"

	"github.com/spf13/cobra"
)

func NewDatabaseCreateCommand(cfg *config.LuetConfig) *cobra.Command {
	var ans = &cobra.Command{
		Use:   "create <artifact_metadata1.yaml> <artifact_metadata1.yaml>",
		Short: "Insert a package in the system DB",
		Long: `Inserts a package in the system database:

		$ luet database create foo.yaml

"luet database create" injects a package in the system database without actually installing it, use it with caution.

This commands takes multiple yaml input file representing package artifacts, that are usually generated while building packages.

The yaml must contain the package definition, and the file list at least.

For reference, inspect a "metadata.yaml" file generated while running "luet build"`,
		Args: cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {

			aManager := installer.NewArtifactsManager(cfg)
			defer aManager.Close()

			aManager.Setup()
			systemDB := aManager.Database

			for _, a := range args {
				dat, err := ioutil.ReadFile(a)
				if err != nil {
					Fatal("Failed reading ", a, ": ", err.Error())
				}
				art, err := artifact.NewPackageArtifactFromYaml(dat)
				if err != nil {
					Fatal("Failed reading yaml ", a, ": ", err.Error())
				}

				files := art.Files

				// Check if the package is already present
				if p, err := systemDB.FindPackage(art.CompileSpec.GetPackage()); err == nil && p.GetName() != "" {
					Fatal("Package", art.CompileSpec.GetPackage().HumanReadableString(),
						" already present.")
				}

				if _, err := systemDB.CreatePackage(art.CompileSpec.GetPackage()); err != nil {
					Fatal("Failed to create ", a, ": ", err.Error())
				}
				if err := systemDB.SetPackageFiles(&pkg.PackageFile{PackageFingerprint: art.CompileSpec.GetPackage().GetFingerPrint(), Files: files}); err != nil {
					Fatal("Failed setting package files for ", a, ": ", err.Error())
				}

				Info(art.CompileSpec.GetPackage().HumanReadableString(), " created")
			}

		},
	}

	return ans
}

/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd_tree

import (
	"fmt"
	"os"

	bhelpers "github.com/macaroni-os/anise/anise-build/cmd/helpers"
	helpers "github.com/macaroni-os/anise/cmd/helpers"
	"github.com/macaroni-os/anise/pkg/compiler"
	"github.com/macaroni-os/anise/pkg/compiler/backend"
	"github.com/macaroni-os/anise/pkg/compiler/types/options"
	. "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"
	pkg "github.com/macaroni-os/anise/pkg/package"
	tree "github.com/macaroni-os/anise/pkg/tree"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type TreePackageResult struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Version  string `json:"version"`
	Path     string `json:"path"`
	Image    string `json:"image"`
}

type TreeResults struct {
	Packages []TreePackageResult `json:"packages"`
}

func NewTreeImageCommand() *cobra.Command {

	var ans = &cobra.Command{
		Use:   "images [OPTIONS]",
		Short: "List of the images of a package",
		PreRun: func(cmd *cobra.Command, args []string) {
			t, _ := cmd.Flags().GetStringArray("tree")
			if len(t) == 0 {
				Fatal("Mandatory tree param missing.")
			}

			if len(args) != 1 {
				Fatal("Expects one package as parameter")
			}
			bhelpers.BindValuesFlags(cmd)
			viper.BindPFlag("image-repository", cmd.Flags().Lookup("image-repository"))

		},
		Run: func(cmd *cobra.Command, args []string) {
			var results TreeResults

			treePath, _ := cmd.Flags().GetStringArray("tree")
			imageRepository := viper.GetString("image-repository")
			pullRepo, _ := cmd.Flags().GetStringArray("pull-repository")
			values := bhelpers.ValuesFlags()

			out, _ := cmd.Flags().GetString("output")
			if out != "terminal" {
				LuetCfg.GetLogging().SetLogLevel("error")
			}

			reciper := tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))

			for _, t := range treePath {
				err := reciper.Load(t)
				if err != nil {
					Fatal("Error on load tree ", err)
				}
			}
			compilerBackend := backend.NewSimpleDockerBackend()

			opts := *LuetCfg.GetSolverOptions()
			luetCompiler := compiler.NewLuetCompiler(
				compilerBackend,
				reciper.GetDatabase(),
				options.WithBuildValues(values),
				options.WithPushRepository(imageRepository),
				options.WithPullRepositories(pullRepo),
				options.WithTemplateFolder(bhelpers.TemplateFolders(false, treePath)),
				options.WithSolverOptions(opts),
			)

			a := args[0]

			pack, err := helpers.ParsePackageStr(LuetCfg, a)
			if err != nil {
				Fatal("Invalid package string ", a, ": ", err.Error())
			}

			spec, err := luetCompiler.FromPackage(pack)
			if err != nil {
				Fatal("Error: " + err.Error())
			}

			ht := compiler.NewHashTree(reciper.GetDatabase())
			hashtree, err := ht.Query(luetCompiler, spec)
			if err != nil {
				Fatal("Error: " + err.Error())
			}

			for _, assertion := range hashtree.Solution { //highly dependent on the order

				//buildImageHash := imageRepository + ":" + assertion.Hash.BuildHash
				currentPackageImageHash := imageRepository + ":" + assertion.Hash.PackageHash

				results.Packages = append(results.Packages, TreePackageResult{
					Name:     assertion.Package.GetName(),
					Version:  assertion.Package.GetVersion(),
					Category: assertion.Package.GetCategory(),
					Image:    currentPackageImageHash,
				})
			}

			y, err := yaml.Marshal(results)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return
			}
			switch out {
			case "yaml":
				fmt.Println(string(y))
			case "json":
				j2, err := yaml.YAMLToJSON(y)
				if err != nil {
					fmt.Printf("err: %v\n", err)
					return
				}
				fmt.Println(string(j2))
			default:
				for _, p := range results.Packages {
					fmt.Println(fmt.Sprintf("%s/%s-%s: %s", p.Category, p.Name, p.Version, p.Image))
				}
			}
		},
	}
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	ans.Flags().StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")
	ans.Flags().StringArrayP("tree", "t", []string{path}, "Path of the tree to use.")
	ans.Flags().String("image-repository", "luet/cache", "Default base image string for generated image")
	ans.Flags().StringArrayP("pull-repository", "p", []string{}, "A list of repositories to pull the cache from")

	return ans
}

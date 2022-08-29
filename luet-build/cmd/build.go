/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	helpers "github.com/geaaru/luet/cmd/helpers"
	"github.com/geaaru/luet/cmd/util"
	"github.com/geaaru/luet/pkg/compiler"
	"github.com/geaaru/luet/pkg/compiler/types/artifact"
	"github.com/geaaru/luet/pkg/compiler/types/compression"
	"github.com/geaaru/luet/pkg/compiler/types/options"
	compilerspec "github.com/geaaru/luet/pkg/compiler/types/spec"
	cfg "github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	tree "github.com/geaaru/luet/pkg/tree"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

type PackageResult struct {
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Version    string   `json:"version"`
	License    string   `json:"License"`
	Repository string   `json:"repository"`
	Target     string   `json:"target"`
	Hidden     bool     `json:"hidden"`
	Files      []string `json:"files"`
}

type Results struct {
	Packages []PackageResult `json:"packages"`
}

func (r *Results) AddPackage(p *PackageResult) {
	r.Packages = append(r.Packages, *p)
}

func (r PackageResult) String() string {
	return fmt.Sprintf("%s/%s-%s required for %s", r.Category, r.Name, r.Version, r.Target)
}

func newBuildCommand(config *cfg.LuetConfig) *cobra.Command {

	var buildCmd = &cobra.Command{
		Use:   "build <package name> <package name> <package name> ...",
		Short: "build a package or a tree",
		Long: `Builds one or more packages from a tree (current directory is implied):

		$ luet build utils/busybox utils/yq ...

	Builds all packages

		$ luet build --all

	Builds only the leaf packages:

		$ luet build --full

	Build package revdeps:

		$ luet build --revdeps utils/yq

	Build package without dependencies (needs the images already in the host, or either need to be available online):

		$ luet build --nodeps utils/yq ...

	Build packages specifying multiple definition trees:

		$ luet build --tree overlay/path --tree overlay/path2 utils/yq ...
	`, PreRun: func(cmd *cobra.Command, args []string) {
			config.Viper.BindPFlag("tree", cmd.Flags().Lookup("tree"))
			config.Viper.BindPFlag("destination", cmd.Flags().Lookup("destination"))
			config.Viper.BindPFlag("backend", cmd.Flags().Lookup("backend"))
			config.Viper.BindPFlag("privileged", cmd.Flags().Lookup("privileged"))
			config.Viper.BindPFlag("revdeps", cmd.Flags().Lookup("revdeps"))
			config.Viper.BindPFlag("all", cmd.Flags().Lookup("all"))
			config.Viper.BindPFlag("compression", cmd.Flags().Lookup("compression"))
			config.Viper.BindPFlag("nodeps", cmd.Flags().Lookup("nodeps"))
			config.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
			util.BindValuesFlags(cmd)
			config.Viper.BindPFlag("backend-args", cmd.Flags().Lookup("backend-args"))

			config.Viper.BindPFlag("image-repository", cmd.Flags().Lookup("image-repository"))
			config.Viper.BindPFlag("push", cmd.Flags().Lookup("push"))
			config.Viper.BindPFlag("pull", cmd.Flags().Lookup("pull"))
			config.Viper.BindPFlag("wait", cmd.Flags().Lookup("wait"))
			config.Viper.BindPFlag("keep-images", cmd.Flags().Lookup("keep-images"))

			util.BindSolverFlags(cmd)

			config.Viper.BindPFlag("general.show_build_output", cmd.Flags().Lookup("live-output"))
			config.Viper.BindPFlag("backend-args", cmd.Flags().Lookup("backend-args"))

		},
		Run: func(cmd *cobra.Command, args []string) {

			treePaths := config.Viper.GetStringSlice("tree")
			dst := config.Viper.GetString("destination")
			concurrency := config.GetGeneral().Concurrency
			backendType := config.Viper.GetString("backend")
			privileged := config.Viper.GetBool("privileged")
			revdeps := config.Viper.GetBool("revdeps")
			all := config.Viper.GetBool("all")
			compressionType := config.Viper.GetString("compression")
			imageRepository := config.Viper.GetString("image-repository")
			values := util.ValuesFlags()
			wait := config.Viper.GetBool("wait")
			push := config.Viper.GetBool("push")
			pull := config.Viper.GetBool("pull")
			keepImages := config.Viper.GetBool("keep-images")
			nodeps := config.Viper.GetBool("nodeps")
			onlydeps := config.Viper.GetBool("onlydeps")
			onlyTarget, _ := cmd.Flags().GetBool("only-target-package")
			full, _ := cmd.Flags().GetBool("full")
			rebuild, _ := cmd.Flags().GetBool("rebuild")

			var results Results
			backendArgs := config.Viper.GetStringSlice("backend-args")

			out, _ := cmd.Flags().GetString("output")
			if out != "terminal" {
				config.GetLogging().SetLogLevel("error")
			}
			pretend, _ := cmd.Flags().GetBool("pretend")
			fromRepo, _ := cmd.Flags().GetBool("from-repositories")

			compilerSpecs := compilerspec.NewLuetCompilationspecs()
			var db pkg.PackageDatabase

			compilerBackend, err := compiler.NewBackend(backendType)
			helpers.CheckErr(err)

			db = pkg.NewInMemoryDatabase(false)
			defer db.Clean()

			generalRecipe := tree.NewCompilerRecipe(db)

			if fromRepo {
				if err := installer.LoadBuildTree(generalRecipe, db, config); err != nil {
					Warning("errors while loading trees from repositories", err.Error())
				}
			}

			for _, src := range treePaths {
				Info("Loading tree", src)
				helpers.CheckErr(generalRecipe.Load(src))
			}

			Info("Building in", dst)

			opts := util.SetSolverConfig()
			pullRepo, _ := cmd.Flags().GetStringArray("pull-repository")

			config.GetGeneral().ShowBuildOutput = config.Viper.GetBool("general.show_build_output")

			Debug("Solver", opts.CompactString())

			luetCompiler := compiler.NewLuetCompiler(compilerBackend,
				generalRecipe.GetDatabase(),
				options.NoDeps(nodeps),
				options.WithBackendType(backendType),
				options.PushImages(push),
				options.WithBuildValues(values),
				options.WithPullRepositories(pullRepo),
				options.WithPushRepository(imageRepository),
				options.Rebuild(rebuild),
				options.WithTemplateFolder(util.TemplateFolders(fromRepo, treePaths)),
				options.Wait(wait),
				options.OnlyTarget(onlyTarget),
				options.PullFirst(pull),
				options.KeepImg(keepImages),
				options.OnlyDeps(onlydeps),
				options.BackendArgs(backendArgs),
				options.Concurrency(concurrency),
				options.WithCompressionType(compression.Implementation(compressionType)),
			)

			if full {
				specs, err := luetCompiler.FromDatabase(generalRecipe.GetDatabase(), true, dst)
				if err != nil {
					Fatal(err.Error())
				}
				for _, spec := range specs {
					Info(":package: Selecting ", spec.GetPackage().GetName(), spec.GetPackage().GetVersion())

					compilerSpecs.Add(spec)
				}
			} else if !all {
				for _, a := range args {
					pack, err := helpers.ParsePackageStr(a)
					if err != nil {
						Fatal("Invalid package string ", a, ": ", err.Error())
					}

					spec, err := luetCompiler.FromPackage(pack)
					if err != nil {
						Fatal("Error: " + err.Error())
					}

					spec.SetOutputPath(dst)
					compilerSpecs.Add(spec)
				}
			} else {
				w := generalRecipe.GetDatabase().World()

				for _, p := range w {
					spec, err := luetCompiler.FromPackage(p)
					if err != nil {
						Fatal("Error: " + err.Error())
					}
					Info(":package: Selecting ", p.GetName(), p.GetVersion())
					spec.SetOutputPath(dst)
					compilerSpecs.Add(spec)
				}
			}

			var artifact []*artifact.PackageArtifact
			var errs []error
			if revdeps {
				artifact, errs = luetCompiler.CompileWithReverseDeps(privileged, compilerSpecs)

			} else if pretend {
				toCalculate := []*compilerspec.LuetCompilationSpec{}
				if full {
					var err error
					toCalculate, err = luetCompiler.ComputeMinimumCompilableSet(compilerSpecs.All()...)
					if err != nil {
						errs = append(errs, err)
					}
				} else {
					toCalculate = compilerSpecs.All()
				}

				for _, sp := range toCalculate {
					ht := compiler.NewHashTree(generalRecipe.GetDatabase())
					hashTree, err := ht.Query(luetCompiler, sp)
					if err != nil {
						errs = append(errs, err)
					}
					for _, p := range hashTree.Dependencies {
						results.Packages = append(results.Packages,
							PackageResult{
								Name:       p.Package.GetName(),
								Version:    p.Package.GetVersion(),
								Category:   p.Package.GetCategory(),
								Repository: "",
								Hidden:     p.Package.IsHidden(),
								Target:     sp.GetPackage().HumanReadableString(),
							})
					}
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
				case "terminal":
					for _, p := range results.Packages {
						Info(p.String())
					}
				}
			} else {

				artifact, errs = luetCompiler.CompileParallel(privileged, compilerSpecs)
			}
			if len(errs) != 0 {
				for _, e := range errs {
					Error("Error: " + e.Error())
				}
				Fatal("Bailing out")
			}
			for _, a := range artifact {
				Info("Artifact generated:", a.Path)
			}
		},
	}

	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}

	flags := buildCmd.Flags()

	flags.StringSliceP("tree", "t", []string{path}, "Path of the tree to use.")
	flags.String("backend", "docker", "backend used (docker,img)")
	flags.Bool("privileged", true, "Privileged (Keep permissions)")
	flags.Bool("revdeps", false, "Build with revdeps")
	flags.Bool("all", false, "Build all specfiles in the tree")
	flags.Bool("full", false, "Build all packages (optimized)")
	flags.StringSlice("values", []string{}, "Build values file to interpolate with each package")
	flags.StringSliceP("backend-args", "a", []string{}, "Backend args")

	flags.String("destination", filepath.Join(path, "build"), "Destination folder")
	flags.String("compression", "none", "Compression alg: none, gzip, zstd")
	flags.String("image-repository", "luet/cache", "Default base image string for generated image")
	flags.Bool("push", false, "Push images to a hub")
	flags.Bool("pull", false, "Pull images from a hub")
	flags.Bool("wait", false, "Don't build all intermediate images, but wait for them until they are available")
	flags.Bool("keep-images", true, "Keep built docker images in the host")
	flags.Bool("nodeps", false, "Build only the target packages, skipping deps (it works only if you already built the deps locally, or by using --pull) ")
	flags.Bool("onlydeps", false, "Build only package dependencies")
	flags.Bool("only-target-package", false, "Build packages of only the required target. Otherwise builds all the necessary ones not present in the destination")
	flags.String("solver-type", "", "Solver strategy")
	flags.Float32("solver-rate", 0.7, "Solver learning rate")
	flags.Float32("solver-discount", 1.0, "Solver discount rate")
	flags.Int("solver-attempts", 9000, "Solver maximum attempts")
	flags.Bool("solver-concurrent", false, "Use concurrent solver (experimental)")
	flags.Bool("live-output", config.GetGeneral().ShowBuildOutput, "Enable live output of the build phase.")
	flags.Bool("from-repositories", false, "Consume the user-defined repositories to pull specfiles from")
	flags.Bool("rebuild", false, "To combine with --pull. Allows to rebuild the target package even if an image is available, against a local values file")
	flags.Bool("pretend", false, "Just print what packages will be compiled")
	flags.StringArrayP("pull-repository", "p", []string{}, "A list of repositories to pull the cache from")

	flags.StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")

	return buildCmd
}

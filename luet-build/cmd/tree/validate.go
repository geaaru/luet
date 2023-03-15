/*
Copyright © 2020-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd_tree

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"sync"

	helpers "github.com/geaaru/luet/luet-build/cmd/helpers"
	compiler "github.com/geaaru/luet/pkg/compiler"
	sd "github.com/geaaru/luet/pkg/compiler/backend"
	"github.com/geaaru/luet/pkg/compiler/types/options"
	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
	tree "github.com/geaaru/luet/pkg/tree"

	"github.com/spf13/cobra"
)

type ValidateOpts struct {
	WithSolver    bool
	OnlyRuntime   bool
	OnlyBuildtime bool
	RegExcludes   []*regexp.Regexp
	RegMatches    []*regexp.Regexp
	Excludes      []string
	Matches       []string

	// Runtime validate stuff
	RuntimeCacheDeps *pkg.InMemoryDatabase
	RuntimeReciper   *tree.InstallerRecipe

	// Buildtime validate stuff
	BuildtimeCacheDeps *pkg.InMemoryDatabase
	BuildtimeReciper   *tree.CompilerRecipe

	Mutex      sync.Mutex
	BrokenPkgs int
	BrokenDeps int

	Errors []error
}

func (o *ValidateOpts) IncrBrokenPkgs() {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()
	o.BrokenPkgs++
}

func (o *ValidateOpts) IncrBrokenDeps() {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()
	o.BrokenDeps++
}

func (o *ValidateOpts) AddError(err error) {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()
	o.Errors = append(o.Errors, err)
}

func validatePackage(p pkg.Package, checkType string, opts *ValidateOpts, reciper tree.Builder, cacheDeps *pkg.InMemoryDatabase) error {
	var errstr string
	var ans error

	var depSolver solver.PackageSolver

	if opts.WithSolver {
		emptyInstallationDb := pkg.NewInMemoryDatabase(false)
		depSolver = solver.NewSolver(solver.Options{Type: solver.SingleCoreSimple}, pkg.NewInMemoryDatabase(false),
			reciper.GetDatabase(),
			emptyInstallationDb)
	}

	found, err := reciper.GetDatabase().FindPackages(
		&pkg.DefaultPackage{
			Name:     p.GetName(),
			Category: p.GetCategory(),
			Version:  ">=0",
		},
	)

	if err != nil || len(found) < 1 {
		if err != nil {
			errstr = err.Error()
		} else {
			errstr = "No packages"
		}
		Error(fmt.Sprintf("[%9s] %s/%s-%s: Broken. No versions could be found by database %s",
			checkType,
			p.GetCategory(), p.GetName(), p.GetVersion(),
			errstr,
		))

		opts.IncrBrokenDeps()

		return errors.New(
			fmt.Sprintf("[%9s] %s/%s-%s: Broken. No versions could be found by database %s",
				checkType,
				p.GetCategory(), p.GetName(), p.GetVersion(),
				errstr,
			))
	}

	// Ensure that we use the right package from right recipier for deps
	pReciper, err := reciper.GetDatabase().FindPackage(
		&pkg.DefaultPackage{
			Name:     p.GetName(),
			Category: p.GetCategory(),
			Version:  p.GetVersion(),
		},
	)
	if err != nil {
		errstr = fmt.Sprintf("[%9s] %s/%s-%s: Error on retrieve package - %s.",
			checkType,
			p.GetCategory(), p.GetName(), p.GetVersion(),
			err.Error(),
		)
		Error(errstr)

		return errors.New(errstr)
	}
	p = pReciper

	pkgstr := fmt.Sprintf("%s/%s-%s", p.GetCategory(), p.GetName(),
		p.GetVersion())

	validpkg := true

	if len(opts.Matches) > 0 {
		matched := false
		for _, rgx := range opts.RegMatches {
			if rgx.MatchString(pkgstr) {
				matched = true
				break
			}
		}

		if !matched {
			return nil
		}
	}

	if len(opts.Excludes) > 0 {
		excluded := false
		for _, rgx := range opts.RegExcludes {
			if rgx.MatchString(pkgstr) {
				excluded = true
				break
			}
		}

		if excluded {
			return nil
		}
	}

	if checkType == "buildtime" {

		// Retrieve the build specs
		c := compiler.NewLuetCompiler(
			sd.NewSimpleDockerBackend(),
			reciper.GetDatabase(),
			options.Concurrency(2),
		)

		spec, err := c.FromPackage(p)
		if err != nil {
			return errors.New(
				fmt.Sprintf(
					"Error on retrieve build specs for package %s: %s",
					p.HumanReadableString(), err.Error()))
			Error(err.Error())
		}

		valid, err := spec.IsValid()
		if !valid {
			errstr := fmt.Sprintf(
				"For package %s/%s-%s found invalid build.yaml: %s",
				p.GetCategory(), p.GetName(), p.GetVersion(),
				err.Error())

			Error(errstr)

			opts.AddError(errors.New(errstr))

			validpkg = false
		}
	}

	Info(fmt.Sprintf("[%9s] Checking package ", checkType)+
		fmt.Sprintf("%s/%s-%s", p.GetCategory(), p.GetName(), p.GetVersion()),
		"with", len(p.GetRequires()), "dependencies and", len(p.GetConflicts()), "conflicts.")

	processRelations := func(r *pkg.DefaultPackage, idx, tot int, conflict bool) {
		var deps pkg.Packages
		var err error
		if r.IsSelector() {
			deps, err = reciper.GetDatabase().FindPackages(
				&pkg.DefaultPackage{
					Name:     r.GetName(),
					Category: r.GetCategory(),
					Version:  r.GetVersion(),
				},
			)
		} else {
			deps = append(deps, r)
		}

		if err != nil || len(deps) < 1 {

			if conflict {
				Warning(fmt.Sprintf("[%9s] %s/%s-%s: Conflict %s-%s-%s not available. Ignoring.",
					checkType,
					p.GetCategory(), p.GetName(), p.GetVersion(),
					r.GetCategory(), r.GetName(), r.GetVersion(),
				))
				return
			}
			if err != nil {
				errstr = err.Error()
			} else {
				errstr = "No packages"
			}
			Error(fmt.Sprintf("[%9s] %s/%s-%s: Broken Dep %s/%s-%s - %s",
				checkType,
				p.GetCategory(), p.GetName(), p.GetVersion(),
				r.GetCategory(), r.GetName(), r.GetVersion(),
				errstr,
			))

			opts.IncrBrokenDeps()

			ans = errors.New(
				fmt.Sprintf("[%9s] %s/%s-%s: Broken Dep %s/%s-%s - %s",
					checkType,
					p.GetCategory(), p.GetName(), p.GetVersion(),
					r.GetCategory(), r.GetName(), r.GetVersion(),
					errstr))

			validpkg = false

		} else {

			Debug(fmt.Sprintf("[%9s] Find packages for dep", checkType),
				fmt.Sprintf("%s/%s-%s", r.GetCategory(), r.GetName(), r.GetVersion()))

			if opts.WithSolver {

				Info(fmt.Sprintf("[%9s]  :soap: [%2d/%2d] %s/%s-%s: %s/%s-%s",
					checkType,
					idx+1, tot,
					p.GetCategory(), p.GetName(), p.GetVersion(),
					r.GetCategory(), r.GetName(), r.GetVersion(),
				))

				// Check if the solver is already been done for the deep
				_, err := cacheDeps.Get(r.HashFingerprint(""))
				if err == nil {
					Debug(fmt.Sprintf("[%9s]  :direct_hit: Cache Hit for dep", checkType),
						fmt.Sprintf("%s/%s-%s", r.GetCategory(), r.GetName(), r.GetVersion()))
					return
				}

				Spinner(32)
				solution, err := depSolver.Install(pkg.Packages{r})
				ass := solution.SearchByName(r.GetPackageName())
				SpinnerStop()
				if err == nil {
					if ass == nil {

						ans = errors.New(
							fmt.Sprintf("[%9s] %s/%s-%s: solution doesn't retrieve package %s/%s-%s.",
								checkType,
								p.GetCategory(), p.GetName(), p.GetVersion(),
								r.GetCategory(), r.GetName(), r.GetVersion(),
							))

						if LuetCfg.GetGeneral().Debug {
							for idx, pa := range solution {
								fmt.Println(fmt.Sprintf("[%9s] %s/%s-%s: solution %d: %s",
									checkType,
									p.GetCategory(), p.GetName(), p.GetVersion(), idx,
									pa.Package.GetPackageName()))
							}
						}

						Error(ans.Error())
						opts.IncrBrokenDeps()
						validpkg = false
					} else {
						_, err = solution.Order(reciper.GetDatabase(), ass.Package.GetFingerPrint())
					}
				}

				if err != nil {

					Error(fmt.Sprintf("[%9s] %s/%s-%s: solver broken for dep %s/%s-%s - %s",
						checkType,
						p.GetCategory(), p.GetName(), p.GetVersion(),
						r.GetCategory(), r.GetName(), r.GetVersion(),
						err.Error(),
					))

					ans = errors.New(
						fmt.Sprintf("[%9s] %s/%s-%s: solver broken for Dep %s/%s-%s - %s",
							checkType,
							p.GetCategory(), p.GetName(), p.GetVersion(),
							r.GetCategory(), r.GetName(), r.GetVersion(),
							err.Error()))

					opts.IncrBrokenDeps()
					validpkg = false
				}

				// Register the key
				cacheDeps.Set(r.HashFingerprint(""), "1")

			}
		}
	} // end processRelations

	all := p.GetRequires()
	all = append(all, p.GetConflicts()...)
	nTot := len(all)
	all = nil
	for idx, r := range p.GetRequires() {
		processRelations(r, idx, nTot, false)
	}

	for idx, r := range p.GetConflicts() {
		processRelations(r, idx, nTot, true)
	}

	if !validpkg {
		opts.IncrBrokenPkgs()
	}

	return ans
}

func validateWorker(i int,
	wg *sync.WaitGroup,
	c <-chan pkg.Package,
	opts *ValidateOpts) {

	defer wg.Done()

	for p := range c {

		if opts.OnlyBuildtime {
			// Check buildtime compiler/deps
			err := validatePackage(p, "buildtime", opts, opts.BuildtimeReciper, opts.BuildtimeCacheDeps)
			if err != nil {
				opts.AddError(err)
				continue
			}
		} else if opts.OnlyRuntime {

			// Check runtime installer/deps
			err := validatePackage(p, "runtime", opts, opts.RuntimeReciper, opts.RuntimeCacheDeps)
			if err != nil {
				opts.AddError(err)
				continue
			}

		} else {

			// Check runtime installer/deps
			err := validatePackage(p, "runtime", opts, opts.RuntimeReciper, opts.RuntimeCacheDeps)
			if err != nil {
				opts.AddError(err)
				continue
			}

			// Check buildtime compiler/deps
			err = validatePackage(p, "buildtime", opts, opts.BuildtimeReciper, opts.BuildtimeCacheDeps)
			if err != nil {
				opts.AddError(err)
			}

		}

	}
}

func initOpts(opts *ValidateOpts, onlyRuntime, onlyBuildtime, withSolver bool, treePaths []string) {
	var err error

	opts.OnlyBuildtime = onlyBuildtime
	opts.OnlyRuntime = onlyRuntime
	opts.WithSolver = withSolver
	opts.RuntimeReciper = nil
	opts.BuildtimeReciper = nil
	opts.BrokenPkgs = 0
	opts.BrokenDeps = 0

	if onlyBuildtime {
		opts.BuildtimeReciper = (tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))).(*tree.CompilerRecipe)
	} else if onlyRuntime {
		opts.RuntimeReciper = (tree.NewInstallerRecipe(pkg.NewInMemoryDatabase(false))).(*tree.InstallerRecipe)
	} else {
		opts.BuildtimeReciper = (tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))).(*tree.CompilerRecipe)
		opts.RuntimeReciper = (tree.NewInstallerRecipe(pkg.NewInMemoryDatabase(false))).(*tree.InstallerRecipe)
	}

	opts.RuntimeCacheDeps = pkg.NewInMemoryDatabase(false).(*pkg.InMemoryDatabase)
	opts.BuildtimeCacheDeps = pkg.NewInMemoryDatabase(false).(*pkg.InMemoryDatabase)

	for _, treePath := range treePaths {
		Info(fmt.Sprintf("Loading :deciduous_tree: %s...", treePath))
		if opts.BuildtimeReciper != nil {
			err = opts.BuildtimeReciper.Load(treePath)
			if err != nil {
				Fatal("Error on load tree ", err)
			}
		}
		if opts.RuntimeReciper != nil {
			err = opts.RuntimeReciper.Load(treePath)
			if err != nil {
				Fatal("Error on load tree ", err)
			}
		}
	}

	opts.RegExcludes, err = helpers.CreateRegexArray(opts.Excludes)
	if err != nil {
		Fatal(err.Error())
	}
	opts.RegMatches, err = helpers.CreateRegexArray(opts.Matches)
	if err != nil {
		Fatal(err.Error())
	}

}

func NewTreeValidateCommand() *cobra.Command {
	var excludes []string
	var matches []string
	var treePaths []string
	var opts ValidateOpts

	var ans = &cobra.Command{
		Use:   "validate [OPTIONS]",
		Short: "Validate a tree or a list of packages",
		Args:  cobra.OnlyValidArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			onlyRuntime, _ := cmd.Flags().GetBool("only-runtime")
			onlyBuildtime, _ := cmd.Flags().GetBool("only-buildtime")

			if len(treePaths) < 1 {
				Fatal("Mandatory tree param missing.")
			}
			if onlyRuntime && onlyBuildtime {
				Fatal("Both --only-runtime and --only-buildtime options are not possibile.")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			var reciper tree.Builder

			concurrency := LuetCfg.GetGeneral().Concurrency

			withSolver, _ := cmd.Flags().GetBool("with-solver")
			onlyRuntime, _ := cmd.Flags().GetBool("only-runtime")
			onlyBuildtime, _ := cmd.Flags().GetBool("only-buildtime")

			opts.Excludes = excludes
			opts.Matches = matches
			initOpts(&opts, onlyRuntime, onlyBuildtime, withSolver, treePaths)

			// We need at least one valid reciper for get list of the packages.
			if onlyBuildtime {
				reciper = opts.BuildtimeReciper
			} else {
				reciper = opts.RuntimeReciper
			}

			all := make(chan pkg.Package)

			var wg = new(sync.WaitGroup)

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go validateWorker(i, wg, all, &opts)
			}
			for _, p := range reciper.GetDatabase().World() {
				all <- p
			}
			close(all)

			wg.Wait()

			stringerrs := []string{}
			for _, e := range opts.Errors {
				stringerrs = append(stringerrs, e.Error())
			}
			sort.Strings(stringerrs)
			for _, e := range stringerrs {
				fmt.Println(e)
			}

			// fmt.Println("Broken packages:", brokenPkgs, "(", brokenDeps, "deps ).")
			if len(stringerrs) != 0 {
				Error(fmt.Sprintf("Found %d broken packages and %d broken deps.",
					opts.BrokenPkgs, opts.BrokenDeps))
				Fatal("Errors: " + strconv.Itoa(len(stringerrs)))
			} else {
				Info("All good! :white_check_mark:")
				os.Exit(0)
			}
		},
	}
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	ans.Flags().Bool("only-runtime", false, "Check only runtime dependencies.")
	ans.Flags().Bool("only-buildtime", false, "Check only buildtime dependencies.")
	ans.Flags().BoolP("with-solver", "s", false,
		"Enable check of requires also with solver.")
	ans.Flags().StringSliceVarP(&treePaths, "tree", "t", []string{path},
		"Path of the tree to use.")
	ans.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{},
		"Exclude matched packages from analysis. (Use string as regex).")
	ans.Flags().StringSliceVarP(&matches, "matches", "m", []string{},
		"Analyze only matched packages. (Use string as regex).")

	return ans
}

// Copyright © 2020 Ettore Di Giacinto <mudler@gentoo.org>
//                  Daniele Rondina <geaaru@sabayonlinux.org>
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

package cmd_tree

import (
	"fmt"
	"os"
	"sort"

	//. "github.com/geaaru/luet/pkg/config"
	"github.com/ghodss/yaml"
	helpers "github.com/geaaru/luet/cmd/helpers"
	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
	tree "github.com/geaaru/luet/pkg/tree"

	"github.com/spf13/cobra"
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

func pkgDetail(pkg pkg.Package) string {
	ans := fmt.Sprintf(`
  @@ Package: %s/%s-%s
     Description: %s
     License:     %s`,
		pkg.GetCategory(), pkg.GetName(), pkg.GetVersion(),
		pkg.GetDescription(), pkg.GetLicense())

	for idx, u := range pkg.GetURI() {
		if idx == 0 {
			ans += fmt.Sprintf("     URLs:        %s", u)
		} else {
			ans += fmt.Sprintf("                  %s", u)
		}
	}

	return ans
}

func NewTreePkglistCommand() *cobra.Command {
	var excludes []string
	var matches []string

	var ans = &cobra.Command{
		Use:   "pkglist [OPTIONS]",
		Short: "List of the packages found in tree.",
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, args []string) {
			t, _ := cmd.Flags().GetStringArray("tree")
			if len(t) == 0 {
				Fatal("Mandatory tree param missing.")
			}

			revdeps, _ := cmd.Flags().GetBool("revdeps")
			deps, _ := cmd.Flags().GetBool("deps")
			if revdeps && deps {
				Fatal("Both revdeps and deps option used. Choice only one.")
			}

		},
		Run: func(cmd *cobra.Command, args []string) {
			var results TreeResults
			var depSolver solver.PackageSolver

			treePath, _ := cmd.Flags().GetStringArray("tree")
			verbose, _ := cmd.Flags().GetBool("verbose")
			buildtime, _ := cmd.Flags().GetBool("buildtime")
			full, _ := cmd.Flags().GetBool("full")
			revdeps, _ := cmd.Flags().GetBool("revdeps")
			deps, _ := cmd.Flags().GetBool("deps")

			out, _ := cmd.Flags().GetString("output")
			if out != "terminal" {
				LuetCfg.GetLogging().SetLogLevel("error")
			}

			var reciper tree.Builder
			if buildtime {
				reciper = tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))
			} else {
				reciper = tree.NewInstallerRecipe(pkg.NewInMemoryDatabase(false))
			}

			for _, t := range treePath {
				err := reciper.Load(t)
				if err != nil {
					Fatal("Error on load tree ", err)
				}
			}

			if deps {
				emptyInstallationDb := pkg.NewInMemoryDatabase(false)

				depSolver = solver.NewSolver(solver.Options{Type: solver.SingleCoreSimple}, pkg.NewInMemoryDatabase(false),
					reciper.GetDatabase(),
					emptyInstallationDb)

			}

			regExcludes, err := helpers.CreateRegexArray(excludes)
			if err != nil {
				Fatal(err.Error())
			}
			regMatches, err := helpers.CreateRegexArray(matches)
			if err != nil {
				Fatal(err.Error())
			}

			plist := make([]string, 0)
			for _, p := range reciper.GetDatabase().World() {
				pkgstr := ""
				addPkg := true
				if full {
					pkgstr = pkgDetail(p)
				} else if verbose {
					pkgstr = p.HumanReadableString()
				} else {
					pkgstr = fmt.Sprintf("%s/%s", p.GetCategory(), p.GetName())
				}

				if len(matches) > 0 {
					matched := false
					for _, rgx := range regMatches {
						if rgx.MatchString(pkgstr) {
							matched = true
							break
						}
					}
					if !matched {
						addPkg = false
					}
				}

				if len(excludes) > 0 && addPkg {
					for _, rgx := range regExcludes {
						if rgx.MatchString(pkgstr) {
							addPkg = false
							break
						}
					}
				}

				if !addPkg {
					continue
				}

				if revdeps {
					packs, _ := reciper.GetDatabase().GetRevdeps(p)
					for i := range packs {
						revdep := packs[i]
						if full {
							pkgstr = pkgDetail(revdep)
						} else if verbose {
							pkgstr = revdep.HumanReadableString()
						} else {
							pkgstr = fmt.Sprintf("%s/%s", revdep.GetCategory(), revdep.GetName())
						}
						plist = append(plist, pkgstr)
						results.Packages = append(results.Packages, TreePackageResult{
							Name:     revdep.GetName(),
							Version:  revdep.GetVersion(),
							Category: revdep.GetCategory(),
							Path:     revdep.GetPath(),
						})
					}
				} else if deps {

					solution, err := depSolver.Install(pkg.Packages{p})
					if err != nil {
						Fatal(err.Error())
					}
					ass := solution.SearchByName(p.GetPackageName())
					solution, err = solution.Order(reciper.GetDatabase(), ass.Package.GetFingerPrint())
					if err != nil {
						Fatal(err.Error())
					}

					for _, pa := range solution {

						if pa.Value {
							// Exclude itself
							if pa.Package.GetName() == p.GetName() && pa.Package.GetCategory() == p.GetCategory() {
								continue
							}

							if full {
								pkgstr = pkgDetail(pa.Package)
							} else if verbose {
								pkgstr = pa.Package.HumanReadableString()
							} else {
								pkgstr = fmt.Sprintf("%s/%s", pa.Package.GetCategory(), pa.Package.GetName())
							}
							plist = append(plist, pkgstr)
							results.Packages = append(results.Packages, TreePackageResult{
								Name:     pa.Package.GetName(),
								Version:  pa.Package.GetVersion(),
								Category: pa.Package.GetCategory(),
								Path:     pa.Package.GetPath(),
							})
						}

					}

				} else {

					plist = append(plist, pkgstr)
					results.Packages = append(results.Packages, TreePackageResult{
						Name:     p.GetName(),
						Version:  p.GetVersion(),
						Category: p.GetCategory(),
						Path:     p.GetPath(),
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
			default:
				if !deps {
					sort.Strings(plist)
				}
				for _, p := range plist {
					fmt.Println(p)
				}
			}

		},
	}
	path, err := os.Getwd()
	if err != nil {
		Fatal(err)
	}
	ans.Flags().BoolP("buildtime", "b", false, "Build time match")
	ans.Flags().StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")
	ans.Flags().Bool("revdeps", false, "Search package reverse dependencies")
	ans.Flags().Bool("deps", false, "Search package dependencies")

	ans.Flags().BoolP("verbose", "v", false, "Add package version")
	ans.Flags().BoolP("full", "f", false, "Show package detail")
	ans.Flags().StringArrayP("tree", "t", []string{path}, "Path of the tree to use.")
	ans.Flags().StringSliceVarP(&matches, "matches", "m", []string{},
		"Include only matched packages from list. (Use string as regex).")
	ans.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{},
		"Exclude matched packages from list. (Use string as regex).")

	return ans
}

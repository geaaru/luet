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
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/jedib0t/go-pretty/table"
	"github.com/geaaru/luet/cmd/util"
	. "github.com/geaaru/luet/pkg/config"
	installer "github.com/geaaru/luet/pkg/installer"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/solver"
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

func searchLocally(term string, results *Results, label, labelMatch, revdeps, hidden bool) {

	system := &installer.System{
		Database: LuetCfg.GetSystemDB(),
		Target:   LuetCfg.GetSystem().Rootfs,
	}

	var err error
	iMatches := pkg.Packages{}
	if label {
		iMatches, err = system.Database.FindPackageLabel(term)
	} else if labelMatch {
		iMatches, err = system.Database.FindPackageLabelMatch(term)
	} else {
		iMatches, err = system.Database.FindPackageMatch(term)
	}

	if err != nil {
		Fatal("Error: " + err.Error())
	}

	for _, pack := range iMatches {
		if !revdeps {
			if !pack.IsHidden() || pack.IsHidden() && hidden {
				f, _ := system.Database.GetPackageFiles(pack)
				results.AddPackage(
					&PackageResult{
						Name:       pack.GetName(),
						Version:    pack.GetVersion(),
						Category:   pack.GetCategory(),
						License:    pack.GetLicense(),
						Repository: "system",
						Hidden:     pack.IsHidden(),
						Files:      f,
					},
				)
			}
		} else {

			packs, _ := system.Database.GetRevdeps(pack)
			for _, revdep := range packs {
				if !revdep.IsHidden() || revdep.IsHidden() && hidden {
					f, _ := system.Database.GetPackageFiles(revdep)
					results.AddPackage(
						&PackageResult{
							Name:       revdep.GetName(),
							Version:    revdep.GetVersion(),
							Category:   revdep.GetCategory(),
							License:    revdep.GetLicense(),
							Repository: "system",
							Hidden:     revdep.IsHidden(),
							Files:      f,
						},
					)
				}
			}
		}
	}

}
func searchOnline(term string, results *Results, label, labelMatch, revdeps, hidden bool) {
	repos := installer.Repositories{}
	for _, repo := range LuetCfg.SystemRepositories {
		if !repo.Enable {
			continue
		}
		r := installer.NewSystemRepository(repo)
		repos = append(repos, r)
	}

	inst := installer.NewLuetInstaller(
		installer.LuetInstallerOptions{
			Concurrency:   LuetCfg.GetGeneral().Concurrency,
			SolverOptions: *LuetCfg.GetSolverOptions(),
		},
	)
	inst.Repositories(repos)

	synced, err := inst.GetRepositoriesInstances(true)
	if err != nil {
		Fatal("Error: " + err.Error())
	}

	matches := []installer.PackageMatch{}
	if label {
		matches = synced.SearchLabel(term)
	} else if labelMatch {
		matches = synced.SearchLabelMatch(term)
	} else {
		matches = synced.Search(term)
	}

	for _, m := range matches {
		if !revdeps {
			if !m.Package.IsHidden() || m.Package.IsHidden() && hidden {
				r := &PackageResult{
					Name:       m.Package.GetName(),
					Version:    m.Package.GetVersion(),
					Category:   m.Package.GetCategory(),
					License:    m.Package.GetLicense(),
					Repository: m.Repo.GetName(),
					Hidden:     m.Package.IsHidden(),
				}
				if m.Artifact != nil {
					r.Files = m.Artifact.Files
				}
				results.AddPackage(r)
			}
		} else {
			packs, _ := m.Repo.GetTree().GetDatabase().GetRevdeps(m.Package)
			for _, revdep := range packs {
				if !revdep.IsHidden() || revdep.IsHidden() && hidden {
					r := &PackageResult{
						Name:       revdep.GetName(),
						Version:    revdep.GetVersion(),
						Category:   revdep.GetCategory(),
						Repository: m.Repo.GetName(),
						License:    revdep.GetLicense(),
						Hidden:     revdep.IsHidden(),
					}
					if m.Artifact != nil {
						r.Files = m.Artifact.Files
					}
					results.AddPackage(r)
				}
			}
		}
	}
}

func searchLocalFiles(term string, results *Results) {
	Info("--- Search results (" + term + "): ---")

	matches, _ := LuetCfg.GetSystemDB().FindPackageByFile(term)
	for _, pack := range matches {
		f, _ := LuetCfg.GetSystemDB().GetPackageFiles(pack)
		results.AddPackage(
			&PackageResult{
				Name:       pack.GetName(),
				Version:    pack.GetVersion(),
				Category:   pack.GetCategory(),
				Repository: "system",
				Hidden:     pack.IsHidden(),
				License:    pack.GetLicense(),
				Files:      f,
			},
		)
	}
}

func searchFiles(term string, results *Results) {
	repos := installer.Repositories{}
	for _, repo := range LuetCfg.SystemRepositories {
		if !repo.Enable {
			continue
		}
		r := installer.NewSystemRepository(repo)
		repos = append(repos, r)
	}

	inst := installer.NewLuetInstaller(
		installer.LuetInstallerOptions{
			Concurrency:   LuetCfg.GetGeneral().Concurrency,
			SolverOptions: *LuetCfg.GetSolverOptions(),
		},
	)
	inst.Repositories(repos)
	synced, err := inst.GetRepositoriesInstances(true)
	if err != nil {
		Fatal("Error: " + err.Error())
	}

	matches := []installer.PackageMatch{}
	matches = synced.SearchPackages(term, installer.FileSearch)

	for _, m := range matches {
		results.AddPackage(
			&PackageResult{
				Name:       m.Package.GetName(),
				Version:    m.Package.GetVersion(),
				Category:   m.Package.GetCategory(),
				Repository: m.Repo.GetName(),
				Hidden:     m.Package.IsHidden(),
				Files:      m.Artifact.Files,
				License:    m.Package.GetLicense(),
			},
		)
	}
}

var searchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Search packages",
	Long: `Search for installed and available packages
	
To search a package in the repositories:

	$ luet search <regex>

To search a package and display results in a table (wide screens):

	$ luet search --table <regex>

To look into the installed packages:

	$ luet search --installed <regex>

Note: the regex argument is optional, if omitted implies "all"

To search a package by label:

	$ luet search --by-label <label>

or by regex against the label:

	$ luet search --by-label-regex <label>

It can also show a package revdeps by:

	$ luet search --revdeps <regex>

Search can also return results in the terminal in different ways: as terminal output, as json or as yaml.

	$ luet search --json <regex> # JSON output
	$ luet search --yaml <regex> # YAML output
`,
	Aliases: []string{"s"},
	PreRun: func(cmd *cobra.Command, args []string) {
		util.BindSystemFlags(cmd)
		util.BindSolverFlags(cmd)
		LuetCfg.Viper.BindPFlag("installed", cmd.Flags().Lookup("installed"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var results Results
		if len(args) > 1 {
			Fatal("Wrong number of arguments (expected 1)")
		} else if len(args) == 0 {
			args = []string{"."}
		}
		hidden, _ := cmd.Flags().GetBool("hidden")

		installed := LuetCfg.Viper.GetBool("installed")
		searchWithLabel, _ := cmd.Flags().GetBool("by-label")
		searchWithLabelMatch, _ := cmd.Flags().GetBool("by-label-regex")
		revdeps, _ := cmd.Flags().GetBool("revdeps")
		tableMode, _ := cmd.Flags().GetBool("table")
		files, _ := cmd.Flags().GetBool("files")

		util.SetSystemConfig()
		util.SetSolverConfig()

		out, _ := cmd.Flags().GetString("output")
		LuetCfg.GetLogging().SetLogLevel("error")

		switch {
		case files && installed:
			searchLocalFiles(args[0], &results)
		case files && !installed:
			searchFiles(args[0], &results)
		case !installed:
			searchOnline(args[0], &results, searchWithLabel,
				searchWithLabelMatch, revdeps, hidden)
		default:
			searchLocally(args[0], &results, searchWithLabel,
				searchWithLabelMatch, revdeps, hidden)
		}

		if out == "json" || out == "yaml" {
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
			}
		} else if tableMode {

			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.AppendHeader(
				table.Row{
					"Package", "Version", "Repository", "License",
				},
			)

			for _, p := range results.Packages {
				t.AppendRow([]interface{}{
					fmt.Sprintf("%s/%s", p.Category, p.Name),
					p.Version,
					p.Repository,
					p.License,
				})
			}
			t.Render()
		} else {
			for _, p := range results.Packages {
				fmt.Println(fmt.Sprintf("%s/%s-%s",
					p.Category, p.Name, p.Version,
				))
			}
		}

	},
}

func init() {
	searchCmd.Flags().String("system-dbpath", "", "System db path")
	searchCmd.Flags().String("system-target", "", "System rootpath")
	searchCmd.Flags().String("system-engine", "", "System DB engine")

	searchCmd.Flags().Bool("installed", false, "Search between system packages")
	searchCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+solver.AvailableResolvers+" )")
	searchCmd.Flags().StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")
	searchCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	searchCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	searchCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	searchCmd.Flags().Bool("by-label", false, "Search packages through label")
	searchCmd.Flags().Bool("by-label-regex", false, "Search packages through label regex")
	searchCmd.Flags().Bool("revdeps", false, "Search package reverse dependencies")
	searchCmd.Flags().Bool("hidden", false, "Include hidden packages")
	searchCmd.Flags().Bool("table", false, "show output in a table (wider screens)")
	searchCmd.Flags().Bool("files", false, "Search between packages files")

	RootCmd.AddCommand(searchCmd)
}

/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_tree

import (
	"fmt"
	"os"
	"sort"

	helpers "github.com/geaaru/luet/luet-build/cmd/helpers"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	tree "github.com/geaaru/luet/pkg/v2/tree"

	"github.com/spf13/cobra"
)

func NewTreePkglistCommand(config *cfg.LuetConfig) *cobra.Command {
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
		},
		Run: func(cmd *cobra.Command, args []string) {
			treePaths, _ := cmd.Flags().GetStringArray("tree")
			verbose, _ := cmd.Flags().GetBool("verbose")

			out, _ := cmd.Flags().GetString("output")
			if out != "terminal" {
				config.GetLogging().SetLogLevel("error")
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
			tResult := tree.NewTreeIdx("", false)

			// Parse indexes
			for _, t := range treePaths {
				ti := tree.NewTreeIdx(t, true).DetectMode()
				err := ti.Read(t)
				if err != nil {
					fmt.Println("Error on read tree " + t + ": " + err.Error())
					os.Exit(1)
				}

				for k, v := range ti.Map {
					for idx := range v {

						pkgstr := ""
						addPkg := true

						if verbose {
							pkgstr = fmt.Sprintf("%s-%s", k, v[idx].Version)
						} else {
							pkgstr = k
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

						plist = append(plist, pkgstr)
						tResult.AddPackage(k, v[idx])
					}

				}
			}

			switch out {
			case "yaml":
				data, err := tResult.ToYAML()
				if err != nil {
					fmt.Println("Error on convert tree in YAML: " + err.Error())
					os.Exit(1)
				}
				fmt.Println(string(data))
			case "json":
				data, err := tResult.ToJSON()
				if err != nil {
					fmt.Println("Error on convert tree in JSON: " + err.Error())
					os.Exit(1)
				}

				fmt.Println(string(data))
			default:
				sort.Strings(plist)
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
	ans.Flags().StringP("output", "o", "terminal", "Output format ( Defaults: terminal, available: json,yaml )")
	ans.Flags().BoolP("verbose", "v", false, "Add package version")
	ans.Flags().StringArrayP("tree", "t", []string{path}, "Path of the tree to use.")
	ans.Flags().StringSliceVarP(&matches, "matches", "m", []string{},
		"Include only matched packages from list. (Use string as regex).")
	ans.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{},
		"Exclude matched packages from list. (Use string as regex).")

	return ans
}

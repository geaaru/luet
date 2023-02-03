/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"errors"
	"path/filepath"
	"regexp"

	"github.com/geaaru/luet/luet-build/pkg/installer"
	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/config"
	"github.com/spf13/cobra"
)

func BindValuesFlags(cmd *cobra.Command) {
	LuetCfg.Viper.BindPFlag("values", cmd.Flags().Lookup("values"))
}

func ValuesFlags() []string {
	return LuetCfg.Viper.GetStringSlice("values")
}

// TemplateFolders returns the default folders which holds shared template between packages in a given tree path
func TemplateFolders(fromRepo bool, treePaths []string) []string {
	templateFolders := []string{}
	if !fromRepo {
		for _, t := range treePaths {
			templateFolders = append(templateFolders, filepath.Join(t, "templates"))
		}
	} else {
		for _, s := range installer.SystemRepositories(LuetCfg) {
			templateFolders = append(templateFolders, filepath.Join(s.TreePath, "templates"))
		}
	}
	return templateFolders
}

func CreateRegexArray(rgx []string) ([]*regexp.Regexp, error) {
	ans := make([]*regexp.Regexp, len(rgx))
	if len(rgx) > 0 {
		for idx, reg := range rgx {
			re := regexp.MustCompile(reg)
			if re == nil {
				return nil, errors.New("Invalid regex " + reg + "!")
			}
			ans[idx] = re
		}
	}

	return ans, nil
}

func BindSolverFlags(cmd *cobra.Command) {
	LuetCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
	LuetCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
	LuetCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
	LuetCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
	LuetCfg.Viper.BindPFlag("solver.implementation", cmd.Flags().Lookup("solver-implementation"))
}

func SetSolverConfig() (c *config.LuetSolverOptions) {
	stype := LuetCfg.Viper.GetString("solver.type")
	discount := LuetCfg.Viper.GetFloat64("solver.discount")
	rate := LuetCfg.Viper.GetFloat64("solver.rate")
	attempts := LuetCfg.Viper.GetInt("solver.max_attempts")
	implementation := LuetCfg.Viper.GetString("solver.implementation")

	LuetCfg.GetSolverOptions().Type = stype
	LuetCfg.GetSolverOptions().LearnRate = float32(rate)
	LuetCfg.GetSolverOptions().Discount = float32(discount)
	LuetCfg.GetSolverOptions().MaxAttempts = attempts
	LuetCfg.GetSolverOptions().Implementation = implementation

	if implementation == "" {
		// Using solver.type until i will drop solver.implementation option.
		LuetCfg.GetSolverOptions().Implementation = stype
		implementation = stype
	}

	return &config.LuetSolverOptions{
		Type:           stype,
		LearnRate:      float32(rate),
		Discount:       float32(discount),
		MaxAttempts:    attempts,
		Implementation: implementation,
	}
}

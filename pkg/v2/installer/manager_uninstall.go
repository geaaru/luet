/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	repos "github.com/geaaru/luet/pkg/v2/repository"
	"github.com/logrusorgru/aurora"

	"github.com/pkg/errors"
)

type UninstallOpts struct {
	Force                       bool
	NoDeps                      bool
	PreserveSystemEssentialData bool
	Ask                         bool
	SkipFinalizers              bool
}

func (m *ArtifactsManager) showPkgs2Remove(list *[]*pkg.DefaultPackage) {
	n := len(*list)

	for idx, p := range *list {
		repos := "::"
		if p.GetRepository() != "" {
			repos += p.GetRepository()
		} else {
			repos = ""
		}
		InfoC(fmt.Sprintf(":knife:[%s of %s] [%s] %-61s - %s",
			aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
			aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", n))),
			aurora.Bold(aurora.BrightYellow("D")),
			aurora.Bold(aurora.BrightYellow(
				fmt.Sprintf("%s%s", p.PackageName(), repos))),
			aurora.Bold(aurora.BrightYellow(p.GetVersion())),
		))
	}
}

func (m *ArtifactsManager) Uninstall(opts *UninstallOpts, targetRootfs string, packs ...*pkg.DefaultPackage) error {
	m.Setup()

	var lastErr error = nil

	matchedPkgs := []*pkg.DefaultPackage{}

	// Check if all packages are availables
	for _, p := range packs {
		if packs, _ := m.Database.FindPackages(p); len(packs) == 0 {
			if opts.Force {
				Warning(fmt.Sprintf(
					"Package %s not found in the system", p.HumanReadableString(),
				))
			} else {
				return errors.New(fmt.Sprintf(
					"Package %s not found in the system", p.HumanReadableString(),
				))
			}
		} else {
			matchedPkgs = append(matchedPkgs, packs[0].(*pkg.DefaultPackage))
		}
	}

	var pkgs2remove []*pkg.DefaultPackage

	if len(matchedPkgs) > 0 {
		if opts.NoDeps {

			m.showPkgs2Remove(&matchedPkgs)

			if opts.Ask {
				if !Ask() {
					return errors.New("Packages remove cancelled by user.")
				}
			}

			pkgs2remove = matchedPkgs

		} else {

			// TODO: temporary load in memory all installed packages.
			systemPkgs := m.Database.World()

			task := NewResolveRdependsTask()
			task.System = &systemPkgs

			for _, p := range matchedPkgs {
				task.Package = p
				err := m.ResolveRuntime(task)
				if err != nil {
					return err
				}
			}

			m.showPkgs2Remove(&task.Matches)

			if opts.Ask {
				if !Ask() {
					return errors.New("Packages remove cancelled by user.")
				}
			}

			pkgs2remove = task.Matches
		}

	}

	// TODO: parallelize this steps. does we need this?
	nPkgs := len(pkgs2remove)
	for idx, p := range pkgs2remove {

		stone := &repos.Stone{
			Name:        p.GetName(),
			Category:    p.GetCategory(),
			Version:     p.GetVersion(),
			Annotations: p.GetAnnotations(),
			Repository:  p.GetRepository(),
		}

		repos := ""
		if stone.Repository != "" {
			repos = "::" + stone.Repository
		}

		msg := fmt.Sprintf(
			"[%3d of %3d] %-65s - %-15s",
			aurora.Bold(aurora.BrightMagenta(idx+1)),
			aurora.Bold(aurora.BrightMagenta(nPkgs)),
			fmt.Sprintf("%s%s", stone.GetName(),
				repos,
			),
			stone.GetVersion())

		err := m.RemovePackage(stone, targetRootfs,
			opts.PreserveSystemEssentialData,
			opts.SkipFinalizers,
			opts.Force,
		)

		if err != nil {
			Error(fmt.Sprintf("[%s] Removing failed: %s",
				stone.HumanReadableString(),
				err.Error(),
			))
			if !opts.Force {
				return err
			} else {
				lastErr = err
			}
		} else {
			Info(fmt.Sprintf(":recycle: %s # uninstalled :check_mark:", msg))
		}
	}

	return lastErr
}

// Retrieve

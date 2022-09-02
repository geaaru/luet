/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"os"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	repos "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/jedib0t/go-pretty/table"
	"github.com/pkg/errors"
)

type UninstallOpts struct {
	Force                       bool
	NoDeps                      bool
	PreserveSystemEssentialData bool
	Ask                         bool
}

func (m *ArtifactsManager) showRemovePkgsTable(list *[]*pkg.DefaultPackage) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(
		table.Row{
			"Package", "Version",
		},
	)

	for _, p := range *list {
		t.AppendRow([]interface{}{
			p.PackageName(), p.GetVersion(),
		})
	}

	t.Render()
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

			m.showRemovePkgsTable(&matchedPkgs)

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

			m.showRemovePkgsTable(&task.Matches)

			if opts.Ask {
				if !Ask() {
					return errors.New("Packages remove cancelled by user.")
				}
			}

			pkgs2remove = task.Matches
		}

	}

	// TODO: parallelize this steps. does we need this?
	for _, p := range pkgs2remove {

		stone := &repos.Stone{
			Name:        p.GetName(),
			Category:    p.GetCategory(),
			Version:     p.GetVersion(),
			Annotations: p.GetAnnotations(),
		}
		err := m.RemovePackage(stone, targetRootfs,
			opts.PreserveSystemEssentialData,
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
		}
	}

	return lastErr
}

// Retrieve

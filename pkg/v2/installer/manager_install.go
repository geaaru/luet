/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"os"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	solver "github.com/geaaru/luet/pkg/v2/solver"

	"github.com/jedib0t/go-pretty/table"
	"github.com/pkg/errors"
)

type InstallOpts struct {
	Force                       bool
	NoDeps                      bool
	PreserveSystemEssentialData bool
	Ask                         bool
	SkipFinalizers              bool
	Pretend                     bool
	DownloadOnly                bool
}

func (m *ArtifactsManager) showPackage2install(
	p2i *artifact.ArtifactsPack,
	p2r *artifact.ArtifactsPack,
) {

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(
		table.Row{
			"Package", "Action", "Version", "Repository", "License",
		},
	)

	if len(p2r.Artifacts) > 0 {
		for _, art := range p2r.Artifacts {
			p := art.GetPackage()
			t.AppendRow([]interface{}{
				p.PackageName(), "D", p.GetVersion(), p.GetRepository(), p.License,
			})
		}
	}

	if len(p2i.Artifacts) > 0 {
		for _, art := range p2i.Artifacts {
			p := art.GetPackage()
			t.AppendRow([]interface{}{
				p.PackageName(), "N", p.GetVersion(), p.GetRepository(), p.License,
			})
		}
	}

	t.SortBy([]table.SortBy{{Name: "Package", Mode: table.Asc}})

	t.Render()
}

func (m *ArtifactsManager) Install(opts *InstallOpts, targetRootfs string,
	packs ...*pkg.DefaultPackage) error {

	errs := []error{}

	m.Setup()

	// TODO: temporary load in memory all installed packages.
	systemPkgs := m.Database.World()

	// Step 1. Check the list of pkgs to install
	//         and exclude packages already installed.
	pkgsToInstall := m._install_s1(opts, &systemPkgs, packs...)
	if len(*pkgsToInstall) == 0 {
		return errors.New("No packages to install.")
	}
	systemPkgs = nil

	// Step 2. Retrieve the last available version of the
	//         selected packages that are admitted by the
	//         existing rootfs packages.
	// Step 3. Check that the selected packages are not
	//         in conflict with new packages.
	// Step 4. Check availability of the required packages.
	//         Wins existing packages. I upgrade deps on
	//         upgrade process only.

	solverOpts := &solver.SolverOpts{
		IgnoreConflicts: false,
		NoDeps:          opts.NoDeps,
	}

	s := solver.NewSolverImplementation("solverv2", m.Config, solverOpts)
	(*s).SetDatabase(m.Database)
	pkgs2Install, pkgs2Remove, err := (*s).Install(packs)
	if err != nil {
		return err
	}
	// Cleanup solver and memory
	s = nil

	if len(pkgs2Install.Artifacts) > 0 {
		m.showPackage2install(pkgs2Install, pkgs2Remove)

		if opts.Pretend {
			return nil
		}

		if opts.Ask {
			if !Ask() {
				return errors.New("Packages install cancelled by user.")
			}
		}

	} else {
		Info("No packages to install.")
	}

	// Step 5. Download all packages to install.
	mapRepos := make(map[string]*wagon.WagonRepository, 0)
	fail := false
	for _, art := range pkgs2Install.Artifacts {
		repoName := art.GetRepository()

		if repoName == "" {
			return fmt.Errorf(
				"Unexpected repository string for package %s",
				art.GetPackage().PackageName())
		}

		var wr *wagon.WagonRepository
		// Create WagonRepository if present
		if _, ok := mapRepos[repoName]; !ok {

			repobasedir := m.Config.GetSystem().GetRepoDatabaseDirPath(repoName)
			repo, err := m.Config.GetSystemRepository(repoName)
			if err != nil {
				Error(
					fmt.Sprintf("Repository not found for artefact %s",
						art.GetPackage().HumanReadableString()))
				fail = true
				continue
			}

			wr = wagon.NewWagonRepository(repo)
			err = wr.ReadWagonIdentify(repobasedir)
			if err != nil {
				fail = true
				Error("Error on read repository identity file: " + err.Error())
				continue
			}

			mapRepos[repoName] = wr
		} else {
			wr = mapRepos[repoName]
		}

		err = m.DownloadPackage(art, wr)
		if err != nil {
			fail = true
			fmt.Println(fmt.Sprintf(
				"Error on download artifact %s: %s",
				art.GetPackage().HumanReadableString(),
				err.Error()))
			Error(fmt.Sprintf("[%40s] download failed :fire:",
				art.GetPackage().HumanReadableString()))
		} else {
			Info(fmt.Sprintf("[%40s] downloaded :check_mark:",
				art.GetPackage().HumanReadableString()))
		}

	}

	if fail {
		return errors.New("Error on download phase.")
	}
	if opts.DownloadOnly {
		return nil
	}

	// Step 6. Order packages.

	// Step 7. Install the matches packages/Remove packages.

	// Run remove of the packages
	if len(pkgs2Remove.Artifacts) > 0 {
		for _, art := range pkgs2Remove.Artifacts {
			p := art.GetPackage()

			stone := &wagon.Stone{
				Name:        p.GetName(),
				Category:    p.GetCategory(),
				Version:     p.GetVersion(),
				Annotations: p.GetAnnotations(),
			}
			err := m.RemovePackage(stone, targetRootfs,
				opts.PreserveSystemEssentialData,
				opts.SkipFinalizers,
				opts.Force,
			)

			if err != nil {
				Error(fmt.Sprintf("[%s] Removing failed: %s",
					stone.HumanReadableString(),
					err.Error()))
				if !opts.Force {
					return err
				} else {
					errs = append(errs, err)
				}
			}

		}
	}

	// Install the new packages
	for _, art := range pkgs2Install.Artifacts {
		art.ResolveCachePath()

		r := mapRepos[art.GetRepository()]

		err = m.InstallPackage(art, r, targetRootfs)
		if err != nil {
			fmt.Println(fmt.Sprintf(
				"Error on install artifact %s: %s",
				art.GetPackage().HumanReadableString(),
				err.Error()))
			Error(fmt.Sprintf("[%40s] install failed - :fire:",
				art.GetPackage().HumanReadableString()))
			return err
		} else {
			Info(fmt.Sprintf("[%40s] installed - :heavy_check_mark:",
				art.GetPackage().HumanReadableString()))
		}

		err = m.RegisterPackage(art, r)
		if err != nil {
			fail = true
			fmt.Println(fmt.Sprintf(
				"Error on register artifact %s: %s",
				art.GetPackage().HumanReadableString(),
				err.Error()))
		}
	}

	// Run finalizers of the installed packages
	for _, art := range pkgs2Install.Artifacts {
		r := mapRepos[art.GetRepository()]

		err = m.ExecuteFinalizer(art, r, true, targetRootfs)
		if err != nil {
			fail = true
		}
	}

	if fail {
		return errors.New("Something goes wrong.")
	}

	return nil
}

func (m *ArtifactsManager) _install_s1(
	opts *InstallOpts, syspkgs *pkg.Packages,
	packs ...*pkg.DefaultPackage) *[]*pkg.DefaultPackage {

	ans := []*pkg.DefaultPackage{}

	sysPkgsMap := syspkgs.ToMap()
	spm := *sysPkgsMap

	for _, p := range packs {
		if _, ok := spm[p.PackageName()]; !ok {
			ans = append(ans, p)
		} else {
			Warning(fmt.Sprintf("[%s] already installed.", p.PackageName()))
		}
	}

	return &ans
}

/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	solver "github.com/geaaru/luet/pkg/v2/solver"
	"github.com/logrusorgru/aurora"

	"github.com/jedib0t/go-pretty/table"
	"github.com/pkg/errors"
)

type InstallOpts struct {
	Force                       bool
	IgnoreConflicts             bool
	NoDeps                      bool
	PreserveSystemEssentialData bool
	Ask                         bool
	SkipFinalizers              bool
	Pretend                     bool
	DownloadOnly                bool
	CheckSystemFiles            bool
	IgnoreMasks                 bool
}

func (m *ArtifactsManager) sortPackages2Install(
	p2i, p2u, p2r *artifact.ArtifactsPack) (*[]*solver.Operation, error) {

	Spinner(3)

	solverOpts := &solver.SolverOpts{
		IgnoreConflicts: false,
		NoDeps:          false,
	}

	s := solver.NewSolverImplementation("solverv2", m.Config, solverOpts)
	(*s).SetDatabase(m.Database)
	ans, err := (*s).OrderOperations(p2i, p2u, p2r)
	SpinnerStop()
	if err != nil {
		return nil, err
	}
	// Cleanup solver and memory
	s = nil

	return ans, nil
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

func (m *ArtifactsManager) ShowReposRevision() error {
	// Show repositories revisions.
	for idx, repo := range m.Config.SystemRepositories {

		if !repo.Enable {
			continue
		}

		repobasedir := m.Config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
		wr := wagon.NewWagonRepository(&m.Config.SystemRepositories[idx])
		err := wr.ReadWagonIdentify(repobasedir)
		if err != nil {
			return fmt.Errorf("Error on read repository identity file: " + err.Error())
		}

		tsec, _ := strconv.ParseInt(wr.Identity.GetLastUpdate(), 10, 64)

		InfoC(
			aurora.Bold(
				aurora.Red(fmt.Sprintf(
					":house:Repository: %30s Revision: ",
					wr.Identity.GetName()))).String() +
				aurora.Bold(aurora.Green(fmt.Sprintf("%3d", wr.GetRevision()))).String() + " - " +
				aurora.Bold(aurora.Green(time.Unix(tsec, 0).String())).String(),
		)
	}

	return nil
}

func (m *ArtifactsManager) Install(opts *InstallOpts, targetRootfs string,
	packs ...*pkg.DefaultPackage) error {

	mapRepos := make(map[string]*wagon.WagonRepository, 0)
	errs := []error{}

	m.Setup()

	err := m.ShowReposRevision()
	if err != nil {
		return err
	}

	// TODO: temporary load in memory all installed packages.
	systemPkgs := m.Database.World()

	// Step 1. Check the list of pkgs to install
	//         and exclude packages already installed.
	pkgsToInstall := m._install_s1(opts, &systemPkgs, packs...)
	if len(*pkgsToInstall) == 0 {
		Warning("No packages to install.")
		return nil
	}
	systemPkgs = nil

	InfoC(":brain:Solving install tree...")
	// Step 2. Retrieve the last available version of the
	//         selected packages that are admitted by the
	//         existing rootfs packages.
	// Step 3. Check that the selected packages are not
	//         in conflict with new packages.
	// Step 4. Check availability of the required packages.
	//         Wins existing packages. I upgrade deps on
	//         upgrade process only.

	Spinner(3)

	solverOpts := &solver.SolverOpts{
		IgnoreConflicts: opts.IgnoreConflicts,
		Force:           opts.Force,
		NoDeps:          opts.NoDeps,
	}

	s := solver.NewSolverImplementation("solverv2", m.Config, solverOpts)
	(*s).SetDatabase(m.Database)
	pkgs2Install, pkgs2Remove, err := (*s).Install(pkgsToInstall)
	SpinnerStop()
	if err != nil {
		return err
	}
	// Cleanup solver and memory
	s = nil

	if len(pkgs2Install.Artifacts) > 0 {
		m.showPackage2install(pkgs2Install, pkgs2Remove)

		err = m.CheckFileConflicts(
			&pkgs2Install.Artifacts,
			&pkgs2Remove.Artifacts,
			opts.CheckSystemFiles, opts.Pretend || opts.Force, targetRootfs,
		)
		if err != nil {
			return err
		}

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
		return nil
	}

	// Step 5. Download all packages to install.
	fail := false
	InfoC(fmt.Sprintf(":truck:Downloading %d packages...",
		len(pkgs2Install.Artifacts)))
	for _, art := range pkgs2Install.Artifacts {
		repoName := art.GetRepository()

		if repoName == "" {
			return fmt.Errorf(
				"Unexpected repository string for package %s",
				art.GetPackage().PackageName())
		}

		var wr *wagon.WagonRepository
		// Create WagonRepository if not present
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
			Error(fmt.Sprintf(":package: %-65s - %-15s # download failed :fire:",
				fmt.Sprintf("%s::%s", art.GetPackage().PackageName(),
					art.GetPackage().Repository,
				),
				art.GetPackage().GetVersion()))
		} else {
			Info(fmt.Sprintf(":package: %-65s - %-15s # downloaded :check_mark:",
				fmt.Sprintf("%s::%s", art.GetPackage().PackageName(),
					art.GetPackage().Repository,
				),
				art.GetPackage().GetVersion()))
		}

	}

	if fail {
		return errors.New("Error on download phase.")
	}
	if opts.DownloadOnly {
		return nil
	}

	InfoC(fmt.Sprintf(":brain:Sorting %d packages operations...",
		len(pkgs2Install.Artifacts)+len(pkgs2Remove.Artifacts)))

	Spinner(3)
	// Step 6. Order packages.
	start := time.Now()
	installOps, err := m.sortPackages2Install(
		pkgs2Install, artifact.NewArtifactsPack(), pkgs2Remove)
	SpinnerStop()
	Debug(fmt.Sprintf(":brain:Sort executed in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))
	if err != nil {
		return err
	}

	InfoC(fmt.Sprintf(
		":clinking_beer_mugs:Executing %d packages operations...",
		len(*installOps),
	))

	// Step 7. Install the matches packages/Remove packages.
	for _, op := range *installOps {

		switch op.Action {
		case solver.RemovePackage:
			p := op.Artifact.GetPackage()

			stone := &wagon.Stone{
				Name:        p.GetName(),
				Category:    p.GetCategory(),
				Version:     p.GetVersion(),
				Annotations: p.GetAnnotations(),
			}
			err = m.RemovePackage(stone, targetRootfs,
				opts.PreserveSystemEssentialData,
				opts.SkipFinalizers,
				opts.Force,
			)

			if err != nil {
				Error(fmt.Sprintf("[%s] Removing failed: %s",
					stone.HumanReadableString(),
					err.Error()))
				fail = true
				if !opts.Force {
					return err
				} else {
					errs = append(errs, err)
				}
			}
		case solver.AddPackage:
			art := op.Artifact
			art.ResolveCachePath()
			r := mapRepos[art.GetRepository()]

			err = m.InstallPackage(art, r, targetRootfs)
			if err != nil {
				Error(fmt.Sprintf(":package: %-65s - %-15s # install failed :fire:",
					fmt.Sprintf("%s::%s", art.GetPackage().PackageName(),
						art.GetPackage().Repository,
					),
					art.GetPackage().GetVersion()))
				errs = append(errs, fmt.Errorf(
					"%s::%s - error: %s", art.GetPackage().PackageName(),
					art.GetPackage().Repository,
					err.Error()))
				fail = true
			} else {
				Info(fmt.Sprintf(":shortcake: %-65s - %-15s # installed :check_mark:",
					fmt.Sprintf("%s::%s", art.GetPackage().PackageName(),
						art.GetPackage().Repository,
					),
					art.GetPackage().GetVersion()))
			}

			err = m.RegisterPackage(art, r, opts.Force)
			if err != nil {
				Error(fmt.Sprintf(
					"Error on register artifact %s: %s",
					art.GetPackage().HumanReadableString(),
					err.Error()))
				fail = true
				if !opts.Force {
					return err
				} else {
					errs = append(errs, err)
				}
			}
		}

	}

	// Run finalizers of the installed packages
	// sorted for action
	if !opts.SkipFinalizers {
		for _, op := range *installOps {
			if op.Action != solver.AddPackage {
				continue
			}
			// POST: just run finalizer on the new packages.
			art := op.Artifact
			r := mapRepos[art.GetRepository()]
			err = m.ExecuteFinalizer(art, r, true, targetRootfs)
			if err != nil {
				fail = true
			}
		}
	}

	if fail {

		// Write all errors again
		if len(errs) > 0 {
			for _, e := range errs {
				Error(e.Error())
			}
		}

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

	// Create a provides map of the installed package.
	// (TODO: Fix this on database. where is used ProvidesDatabase in memory
	//        without a correct setup)
	provMap := make(map[string]*pkg.DefaultPackage, 0)
	for _, p := range *syspkgs {
		if p.HasProvides() {
			for _, pp := range p.GetProvides() {
				provMap[pp.PackageName()] = p.(*pkg.DefaultPackage)
			}
		}
	}

	for _, p := range packs {
		if _, ok := spm[p.PackageName()]; !ok {
			// Check if the package is available as provides
			if prov, ok := provMap[p.PackageName()]; ok {
				Warning(fmt.Sprintf("%s already provided by %s.",
					p.PackageName(), prov.PackageName()))
			} else {
				ans = append(ans, p)
			}
		} else {
			Warning(fmt.Sprintf("%s already installed.",
				aurora.Bold(p.PackageName())))
		}
	}

	return &ans
}

/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"os"
	"time"

	. "github.com/geaaru/luet/pkg/logger"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	solver "github.com/geaaru/luet/pkg/v2/solver"
	"github.com/jedib0t/go-pretty/table"
	"github.com/pkg/errors"
)

func (m *ArtifactsManager) showPackages2Update(
	p2i *artifact.ArtifactsPack,
	p2u *artifact.ArtifactsPack,
	p2r *artifact.ArtifactsPack,
) {

	// NOTE: PRE: For package with p2u I will have a package on p2r too.

	p2umap := p2u.ToMap()
	p2rmap := p2r.ToMap()

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(
		table.Row{
			"Package", "Action", "Version", "Repository", "License",
		},
	)

	if len(p2i.Artifacts) > 0 {
		for _, art := range p2i.Artifacts {
			p := art.GetPackage()
			license := p.License
			if len(license) > 50 {
				license = license[0:47] + "..."
			}
			t.AppendRow([]interface{}{
				p.PackageName(), "N", p.GetVersion(), p.GetRepository(), license,
			})
		}
	}
	if len(p2u.Artifacts) > 0 {
		for pname, art := range p2umap.Artifacts {
			pr, ok := p2rmap.Artifacts[pname]
			p := art[0].GetPackage()
			version := p.GetVersion()
			license := p.License
			repository := p.GetRepository()
			if len(license) > 50 {
				license = license[0:47] + "..."
			}
			if !ok {
				Warning(fmt.Sprintf("For package %s to update not found package to remove",
					pname))
			} else {
				if pr[0].GetPackage().GetVersion() == version {
					version = "*" + version
				} else {
					version = pr[0].GetPackage().GetVersion() + " -> " + version
				}

				if repository != pr[0].GetRepository() {
					if pr[0].GetRepository() == "" {
						repository = "unknown -> " + repository
					} else {
						repository = pr[0].GetRepository() + " -> " + repository
					}
				}
			}

			t.AppendRow([]interface{}{
				p.PackageName(), "U", version, repository, license,
			})
		}
	}

	if len(p2r.Artifacts) > 0 {
		for pname, art := range p2rmap.Artifacts {
			if _, ok := p2umap.Artifacts[pname]; !ok {
				p := art[0].GetPackage()
				license := p.License
				if len(license) > 50 {
					license = license[0:47] + "..."
				}
				t.AppendRow([]interface{}{
					p.PackageName(), "D", p.GetVersion(), p.GetRepository(), license,
				})

			}
		}
	}

	t.SortBy([]table.SortBy{{Name: "Package", Mode: table.Asc}})

	t.Render()
}

func (m *ArtifactsManager) Upgrade(opts *InstallOpts, targetRootfs string) error {
	mapRepos := make(map[string]*wagon.WagonRepository, 0)
	errs := []error{}

	m.Setup()

	err := m.ShowReposRevision()
	if err != nil {
		return err
	}

	Info(":thinking:Computing upgrade, please hang tight... :zzz:")

	Spinner(3)

	solverOpts := &solver.SolverOpts{
		IgnoreConflicts: opts.IgnoreConflicts,
		Force:           opts.Force,
		NoDeps:          opts.NoDeps,
	}

	s := solver.NewSolverImplementation("solverv2", m.Config, solverOpts)
	(*s).SetDatabase(m.Database)
	pkgs2Remove, pkgs2Update, pkgs2Install, err := (*s).Upgrade()
	SpinnerStop()
	if err != nil {
		return err
	}

	if len(pkgs2Remove.Artifacts) == 0 &&
		len(pkgs2Update.Artifacts) == 0 &&
		len(pkgs2Install.Artifacts) == 0 {
		// POST: No new updates.
		InfoC(":smiling_face_with_sunglasses:No packages to updates. The system is updated.")

		return nil

	}

	m.showPackages2Update(pkgs2Install, pkgs2Update, pkgs2Remove)

	iandu := []*artifact.PackageArtifact{}
	iandu = append(iandu, pkgs2Install.Artifacts...)
	iandu = append(iandu, pkgs2Update.Artifacts...)
	err = m.CheckFileConflicts(
		&iandu,
		&pkgs2Remove.Artifacts,
		opts.CheckSystemFiles, opts.Pretend || opts.Force, targetRootfs,
	)
	if err != nil {
		return err
	}
	iandu = nil

	if opts.Pretend {
		return nil
	}

	if opts.Ask {
		if !Ask() {
			return errors.New("Packages upgrade cancelled by user.")
		}
	}

	// Download all packages to install/updates
	fail := false
	InfoC(fmt.Sprintf(":truck:Downloading %d packages...",
		len(pkgs2Install.Artifacts)+len(pkgs2Update.Artifacts)))

	pkgs2Download := append(pkgs2Install.Artifacts, pkgs2Update.Artifacts...)
	for _, art := range pkgs2Download {
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
	pkgs2Download = nil

	if fail {
		return errors.New("Error on download phase.")
	}

	if opts.DownloadOnly {
		return nil
	}

	InfoC(fmt.Sprintf(":brain:Sorting %d packages operations...",
		len(pkgs2Install.Artifacts)+len(pkgs2Update.Artifacts)+len(pkgs2Remove.Artifacts)))

	Spinner(3)
	// Step 6. Order packages.
	start := time.Now()
	installOps, err := m.sortPackages2Install(pkgs2Install, pkgs2Update, pkgs2Remove)
	SpinnerStop()
	Debug(fmt.Sprintf(":brain:Sort executed in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))
	if err != nil {
		return err
	}

	// Cleanup solver and memory
	s = nil

	InfoC(fmt.Sprintf(
		":clinking_beer_mugs:Executing %d packages operations...",
		len(*installOps)))

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
		case solver.AddPackage, solver.UpdatePackage:
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

			err = m.RegisterPackage(art, r)
			if err != nil {
				fail = true
				Error(fmt.Sprintf(
					"Error on register artifact %s: %s",
					art.GetPackage().HumanReadableString(),
					err.Error()))
				errs = append(errs, fmt.Errorf(
					"%s::%s - error: %s", art.GetPackage().PackageName(),
					art.GetPackage().Repository,
					err.Error()))
			}
		}
	}

	// Run finalizers of the installed packages
	// sorted for action
	if !opts.SkipFinalizers {
		for _, op := range *installOps {
			if op.Action == solver.RemovePackage {
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

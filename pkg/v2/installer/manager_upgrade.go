/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"time"

	. "github.com/geaaru/luet/pkg/logger"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	solver "github.com/geaaru/luet/pkg/v2/solver"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
)

func (m *ArtifactsManager) showPackagesSorted(
	p2r *artifact.ArtifactsPack,
	p2u *artifact.ArtifactsPack,
	installOps *[]*solver.Operation,
	withUpdateAndDelete bool) {

	p2rmap := p2r.ToMap()
	p2umap := p2u.ToMap()

	aurora := GetAurora()

	// TODO: Found a more precise
	nOps := len(*installOps)
	if withUpdateAndDelete {
		nOps -= len(p2u.Artifacts)
	}

	idx := 0
	for _, op := range *installOps {
		p := op.Artifact.GetPackage()

		Debug(fmt.Sprintf("Operation %d: %s %s", idx+1, op.Action, p.HumanReadableString()))

		switch op.Action {

		case solver.RemovePackage:

			if _, ok := p2umap.Artifacts[p.PackageName()]; !ok {
				repos := "::"
				if p.GetRepository() != "" {
					repos += p.GetRepository()
				} else {
					repos = ""
				}

				InfoC(fmt.Sprintf(":knife:[%s of %s] [%s] %-61s - %s",
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
					aurora.Bold(aurora.BrightYellow(solver.RemovePackage)),
					aurora.Bold(aurora.BrightYellow(
						fmt.Sprintf("%s%s", p.PackageName(), repos))),
					aurora.Bold(aurora.BrightYellow(p.GetVersion())),
				))
				idx++
			}

		case solver.AddPackage:
			InfoC(fmt.Sprintf(":icecream:[%s of %s] [%s] %-61s - %s",
				aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
				aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
				aurora.Bold(aurora.BrightRed(solver.AddPackage)),
				aurora.Bold(aurora.BrightRed(
					fmt.Sprintf("%s::%s", p.PackageName(),
						p.GetRepository()))),
				aurora.Bold(aurora.BrightRed(p.GetVersion())),
			))
			idx++

		case solver.UpdatePackage:
			pr, ok := p2rmap.Artifacts[p.PackageName()]
			if ok {
				repos := pr[0].GetRepository()
				if repos == "" {
					repos = "unknown"
				}

				version := p.GetVersion()
				if pr[0].GetPackage().GetVersion() == version {
					version = "*" + version
				}

				if repos == p.GetRepository() {
					InfoC(fmt.Sprintf(":cupcake:[%s of %s] [%s] %-61s - %s [%s]",
						aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
						aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
						aurora.Bold(aurora.Green(solver.UpdatePackage)),
						aurora.Bold(
							aurora.Green(
								fmt.Sprintf("%s::%s",
									p.PackageName(), p.GetRepository(),
								),
							),
						),
						aurora.Bold(aurora.Green(version)),
						aurora.BrightCyan(pr[0].GetPackage().GetVersion()),
					))
				} else {
					InfoC(fmt.Sprintf(":candy:[%s of %s] [%s] %-61s - %s [%s]",
						aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
						aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
						aurora.Bold(aurora.Green(solver.UpdatePackage)),
						aurora.Bold(aurora.Green(
							fmt.Sprintf("%s::%s", p.PackageName(), p.GetRepository()),
						)),
						aurora.Bold(aurora.Green(version)),
						aurora.BrightCyan(fmt.Sprintf("%s::%s", pr[0].GetPackage().GetVersion(), repos)),
					))
				}
			} else {
				// POST: Update without remove
				InfoC(fmt.Sprintf(":pie:[%s of %s] [%s] %-61s - %s",
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
					aurora.Bold(aurora.BrightGreen(solver.UpdatePackage)),
					aurora.Bold(aurora.BrightGreen(fmt.Sprintf("%s::%s",
						p.PackageName(), p.GetRepository()),
					)),
					aurora.Bold(aurora.BrightGreen(p.GetVersion())),
				))
			}
			idx++

		case solver.DowngradePackage:
			pr, _ := p2rmap.Artifacts[p.PackageName()]
			repos := pr[0].GetRepository()
			if repos == "" {
				repos = "unknown"
			}

			version := p.GetVersion()

			if repos == p.GetRepository() {
				InfoC(fmt.Sprintf(":doughnut:[%s of %s] [%s] %-61s - %s [%s]",
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
					aurora.Bold(aurora.BrightBlue(solver.DowngradePackage)),
					aurora.Bold(
						aurora.BrightBlue(
							fmt.Sprintf("%s::%s",
								p.PackageName(), p.GetRepository(),
							),
						),
					),
					aurora.Bold(aurora.BrightBlue(version)),
					aurora.BrightCyan(pr[0].GetPackage().GetVersion()),
				))
			} else {
				InfoC(fmt.Sprintf(":lollipop:[%s of %s] [%s] %-61s - %s [%s]",
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", idx+1))),
					aurora.Bold(aurora.BrightMagenta(fmt.Sprintf("%3d", nOps))),
					aurora.Bold(aurora.BrightBlue(solver.DowngradePackage)),
					aurora.Bold(aurora.BrightBlue(
						fmt.Sprintf("%s::%s", p.PackageName(), p.GetRepository()),
					)),
					aurora.Bold(aurora.BrightBlue(version)),
					aurora.BrightCyan(fmt.Sprintf("%s::%s", pr[0].GetPackage().GetVersion(), repos)),
				))
			}
			idx++

		}
	}

}

func (m *ArtifactsManager) showPackages2Update(
	p2i *artifact.ArtifactsPack,
	p2u *artifact.ArtifactsPack,
	p2r *artifact.ArtifactsPack,
) {

	// NOTE: PRE: For package with p2u I will have a package on p2r too.

	p2umap := p2u.ToMap()
	p2rmap := p2r.ToMap()

	// Create operation list sorted by package name

	ops := []*solver.Operation{}

	if len(p2i.Artifacts) > 0 {
		for _, art := range p2i.Artifacts {
			ops = append(ops, &solver.Operation{
				Action:   solver.AddPackage,
				Artifact: art,
			})
		}
	}
	if len(p2u.Artifacts) > 0 {
		for _, art := range p2umap.Artifacts {
			gp, _ := art[0].GetPackage().ToGentooPackage()

			// Check if the package removed has a version
			// greather then the new.
			pr, err := p2rmap.GetArtifactsByKey(
				art[0].GetPackage().PackageName())

			if err == nil {
				gpr, _ := pr[0].GetPackage().ToGentooPackage()
				if val, err := gpr.GreaterThan(gp); err == nil && val {
					ops = append(ops, &solver.Operation{
						Action:   solver.DowngradePackage,
						Artifact: art[0],
					})
					continue
				}
			}

			ops = append(ops, &solver.Operation{
				Action:   solver.UpdatePackage,
				Artifact: art[0],
			})
		}
	}

	if len(p2r.Artifacts) > 0 {
		for pname, art := range p2rmap.Artifacts {
			if _, ok := p2umap.Artifacts[pname]; !ok {
				ops = append(ops, &solver.Operation{
					Action:   solver.RemovePackage,
					Artifact: art[0],
				})
			}
		}
	}

	InfoC(":party_popper:Upgrades:")
	solver.SortOperationsByName(&ops, false)
	m.showPackagesSorted(p2r, p2u, &ops, false)
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
		Deep:            opts.Deep,
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

	if opts.Ask && !opts.ShowInstallOrder {
		if !Ask() {
			return errors.New("Packages upgrade cancelled by user.")
		}
	}

	// Download all packages to install/updates
	fail := false
	InfoC(fmt.Sprintf(":truck:Downloading %d packages...",
		len(pkgs2Install.Artifacts)+len(pkgs2Update.Artifacts)))

	pkgs2Download := append(pkgs2Install.Artifacts, pkgs2Update.Artifacts...)
	ndownloads := len(pkgs2Download)
	for idx, art := range pkgs2Download {
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

		msg := fmt.Sprintf(
			"[%3d of %3d] %-65s - %-15s",
			aurora.Bold(aurora.BrightMagenta(idx+1)),
			aurora.Bold(aurora.BrightMagenta(ndownloads)),
			fmt.Sprintf("%s::%s", art.GetPackage().PackageName(),
				art.GetPackage().Repository,
			),
			art.GetPackage().GetVersion())

		err = m.DownloadPackage(art, wr, msg)
		if err != nil {
			fail = true
			fmt.Println(fmt.Sprintf(
				"Error on download artifact %s: %s",
				art.GetPackage().HumanReadableString(),
				err.Error()))
			Error(fmt.Sprintf(":package:%s # download failed :fire:",
				msg))
		} else {
			Info(fmt.Sprintf(":package:%s # downloaded :check_mark:",
				msg))
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

	if opts.ShowInstallOrder {
		InfoC(":brain:Upgrade order:")
		m.showPackagesSorted(pkgs2Remove, pkgs2Update, installOps, true)
		return nil
	}

	InfoC(fmt.Sprintf(
		":clinking_beer_mugs:Executing %d packages operations...",
		len(*installOps)))
	nOps := len(*installOps)

	for idx, op := range *installOps {

		repos := ""
		if op.Artifact.GetPackage().Repository != "" {
			repos = "::" + op.Artifact.GetPackage().Repository
		}

		msg := fmt.Sprintf(
			"[%3d of %3d] %-65s - %-15s",
			aurora.Bold(aurora.BrightMagenta(idx+1)),
			aurora.Bold(aurora.BrightMagenta(nOps)),
			fmt.Sprintf("%s%s", op.Artifact.GetPackage().PackageName(),
				repos,
			),
			op.Artifact.GetPackage().GetVersion())

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
			} else {
				Info(fmt.Sprintf(":recycle: %s # removed :check_mark:", msg))
			}

		case solver.AddPackage, solver.UpdatePackage, solver.DowngradePackage:
			art := op.Artifact
			art.ResolveCachePath()
			r := mapRepos[art.GetRepository()]

			err = m.InstallPackage(art, r, targetRootfs)
			if err != nil {
				Error(fmt.Sprintf(":package:%s # install failer :fire:", msg))
				errs = append(errs, fmt.Errorf(
					"%s::%s - error: %s", art.GetPackage().PackageName(),
					art.GetPackage().Repository,
					err.Error()))
				fail = true
			} else {
				Info(fmt.Sprintf(":shortcake:%s # installed :check_mark:", msg))
			}

			err = m.RegisterPackage(art, r, opts.Force)
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

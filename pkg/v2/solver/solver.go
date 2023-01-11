/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"errors"
	"fmt"

	"github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
)

type Solver struct {
	Config *config.LuetConfig `yaml:",inline" json:",inline"`
	Opts   *SolverOpts        `yaml:"opts" json:"opts"`

	Database pkg.PackageDatabase `yaml:"-" json:"-"`
	Searcher wagon.Searcher      `yaml:"-" json:"-"`

	conflictsMap     *pkg.PkgsMapList       `yaml:"-" json:"-"`
	systemMap        *pkg.PkgsMapList       `yaml:"-" json:"-"`
	providesMap      *pkg.PkgsMapList       `yaml:"provides,omitempty" json:"provides,omitempty"`
	availableArtsMap *artifact.ArtifactsMap `yaml:"-" json:"-"`
	candidatesMap    *artifact.ArtifactsMap `yaml:"-" json:"-"`
}

func NewSolver(cfg *config.LuetConfig, opts *SolverOpts) *Solver {
	return &Solver{
		Config:        cfg,
		Opts:          opts,
		Database:      nil,
		candidatesMap: artifact.NewArtifactsMap(),
	}
}

func (s *Solver) GetType() SolverType {
	return SingleCoreV3
}

func (s *Solver) SetDatabase(d pkg.PackageDatabase) { s.Database = d }

func (s *Solver) Upgrade() (*artifact.ArtifactsPack, *artifact.ArtifactsPack, *artifact.ArtifactsPack, error) {
	return nil, nil, nil, errors.New("Not yet implemented")
}

func (s *Solver) Install(pkgs pkg.DefaultPackages) (*artifact.ArtifactsPack, *artifact.ArtifactsPack, error) {
	ans2Install := artifact.NewArtifactsPack()
	ans2Remove := artifact.NewArtifactsPack()

	if s.Database == nil {
		return nil, nil, errors.New("Solver Install requires Database")
	}

	// PRE: the input packages are with valid category/name strings.

	searcher := wagon.NewSearcherSimple(s.Config)
	searchOpts := &wagon.StonesSearchOpts{
		Packages:         pkgs,
		Categories:       []string{},
		Labels:           []string{},
		LabelsMatches:    []string{},
		Matches:          []string{},
		FilesOwner:       []string{},
		Annotations:      []string{},
		Hidden:           false,
		AndCondition:     false,
		WithFiles:        true,
		WithRootfsPrefix: false,
		Full:             true,
		OnlyPackages:     true,
	}
	s.Searcher = searcher

	// For every package in list retrieve all available candidates
	// and store the result on ArtifactsMap
	reposArtifacts, err := searcher.SearchArtifacts(searchOpts)
	if err != nil {
		return nil, nil, err
	}

	// Convert the results in a map with all available versions of the
	// selected packages.
	artsPack := &artifact.ArtifactsPack{
		Artifacts: *reposArtifacts,
	}
	s.availableArtsMap = artsPack.ToMap()
	artsPack = nil
	reposArtifacts = nil

	// TODO: Use a different solution with less memory usage
	systemPkgs := s.Database.World()

	s.prepareConflictsAndSystemMap(&systemPkgs)

	// Process all selected packages to install.
	// Created the key list to permit changes on the map
	// meantime that packages are elaborated.
	pList := []string{}
	for pname, _ := range s.availableArtsMap.Artifacts {
		pList = append(pList, pname)
	}

	for _, pname := range pList {
		err := s.resolvePackage(pname, []string{})
		if err != nil {
			return nil, nil, err
		}
	}

	// Cleanup resources
	s.systemMap = nil
	s.conflictsMap = nil
	s.providesMap = nil

	// TODO: sort the packages to install

	// Create the list of package to install
	if s.Opts.NoDeps {
		for _, pkg := range pkgs {
			plist, _ := s.candidatesMap.Artifacts[pkg.PackageName()]
			ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
		}
	} else {
		for pkg, _ := range s.candidatesMap.Artifacts {
			plist, _ := s.candidatesMap.Artifacts[pkg]
			ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
		}
	}

	return ans2Install, ans2Remove, nil
}

func (s *Solver) resolvePackage(pkgstr string, stack []string) error {
	if helpers.ContainsElem(&stack, pkgstr) {
		// POST: this dependency/package is already been elaborated.
		return nil
	}

	_, ok := s.candidatesMap.Artifacts[pkgstr]
	if ok {
		// POST: the package is already been elaborated
		return nil
	}

	// Sort all available versions of the selected package.
	// The first is the major version
	selectedArts, err := s.availableArtsMap.GetSortedArtifactsByKey(pkgstr)
	if err != nil {
		return err
	}

	foundMatched := false

	// NOTE: if the package is not installed in the system
	//       means that there aren't packages that requires it.

	// Map to avoid processing of the same version multiple time.
	bannedVersion := make(map[string]bool, 0)

	for idx, _ := range selectedArts {
		art := selectedArts[idx]
		// If version is already been processed and banned I will
		// skip the artefact
		if _, ok := bannedVersion[art.GetPackage().GetVersion()]; ok {
			continue
		}

		// Check if the selected package is in conflicts with
		// existing tree.
		if !s.Opts.IgnoreConflicts && s.artefactIsInConflict(art) {
			bannedVersion[art.GetPackage().GetVersion()] = true
			continue
		}

		// Validate the selected package with new packages
		// in queue.
		admit, err := s.artefactAdmitByQueue(art)
		if err != nil {
			return err
		}

		if !admit {
			continue
		}

		ss := append(stack, art.GetPackage().PackageName())
		// Check and in queue all package dependencies
		admittedDeps, err := s.processArtefactDeps(art, ss)
		if err != nil {
			return err
		}

		if !admittedDeps {
			// POST: Not all packages dependencies are admit by
			//       the current system.
			continue
		}

		foundMatched = true
		break
	}

	if !foundMatched {
		return fmt.Errorf("No valid candidate found for %s", pkgstr)
	}

	firstValid := false
	// Rebuild the list of available versions to exclude already banned version
	validArts := []*artifact.PackageArtifact{}
	for idx, _ := range selectedArts {
		art := selectedArts[idx]
		if _, ok := bannedVersion[art.GetPackage().GetVersion()]; ok {
			continue
		}
		// Add provides on map
		if !firstValid {
			p := art.GetPackage()
			if len(p.Provides) > 0 {
				for _, prov := range p.Provides {
					s.providesMap.Add(prov.PackageName(), p)
				}
			}
			firstValid = true
		}
		validArts = append(validArts, art)
	}

	s.candidatesMap.Artifacts[pkgstr] = validArts

	return nil
}

func (s *Solver) processArtefactDeps(art *artifact.PackageArtifact, stack []string) (bool, error) {
	candidate := art.GetPackage()

	if len(candidate.PackageRequires) == 0 {
		return true, nil
	}

	for _, p := range candidate.PackageRequires {

		val, ok := s.systemMap.Packages[p.PackageName()]
		if ok {
			// Check if the dependency installed is admitted by the package to install
			admit, err := candidate.Admit(val[0])
			if err != nil {
				return false, err
			} else if !admit {
				return false, nil
			}

			// Nothing to do. The dependency is already on system and is valid.
			continue
		}

		// Check if the dependency is provided.
		provides, ok := s.providesMap.Packages[p.PackageName()]
		if ok {
			provMatched := false
			// Check if the provieded version is admitted by the package
			for _, prov := range provides {
				for idx, _ := range prov.Provides {
					admit, _ := candidate.Admit(prov.Provides[idx])
					if admit {
						provMatched = true
						break
					}
				}
				if provMatched {
					break
				}
			}

			if provMatched {
				// Nothing to do. The dependency is already on system as
				// provides.
				continue
			}
		}

		// POST: the dependency is not installed. Check if already been elaborated
		_, ok = s.candidatesMap.Artifacts[p.PackageName()]
		if ok {
			// TODO: check if the version is valid for the artefact.
			// POST: dependency already added on queue and elaborated. Nothing to do.
			continue
		}

		// POST: check if the dependency is already on stack
		if helpers.ContainsElem(&stack, p.PackageName()) {
			// POST: this dependency/package is under analysis.
			continue
		}

		// Search all availables artefacts from enabled repositories.
		searchOpts := &wagon.StonesSearchOpts{
			Packages: []*pkg.DefaultPackage{
				pkg.NewPackageWithCatThin(p.Category, p.Name, p.Version),
			},
			Categories:       []string{},
			Labels:           []string{},
			LabelsMatches:    []string{},
			Matches:          []string{},
			FilesOwner:       []string{},
			Annotations:      []string{},
			Hidden:           false,
			AndCondition:     false,
			WithFiles:        true,
			WithRootfsPrefix: false,
			Full:             true,
			OnlyPackages:     true,
		}
		Debug(fmt.Sprintf("[%30s] Searching for dependency %s...",
			candidate.PackageName(), searchOpts.Packages[0].PackageName()))
		reposArtifacts, err := s.Searcher.SearchArtifacts(searchOpts)
		if err != nil {
			return false, err
		}

		// Convert the results in a map with all available versions of the
		// selected packages.
		provStr := ""
		for _, depArt := range *reposArtifacts {
			// If the researched package is provided we need to use
			// the name of the package that provides the requirements.
			if provStr == "" &&
				depArt.GetPackage().PackageName() != searchOpts.Packages[0].PackageName() {
				provStr = depArt.GetPackage().PackageName()
			}

			s.availableArtsMap.Add(depArt)
		}

		if provStr != "" {
			err = s.resolvePackage(provStr, stack)
		} else {
			err = s.resolvePackage(p.PackageName(), stack)
		}
		if err != nil {
			return false, err
		}

	} // end for

	return true, nil
}

func (s *Solver) artefactAdmitByQueue(art *artifact.PackageArtifact) (bool, error) {
	// Check if existing conflicts field are in
	// conflicts with the selected artefact
	if len(s.candidatesMap.Artifacts) > 0 {
		for k, _ := range s.candidatesMap.Artifacts {
			artInQueue := s.candidatesMap.Artifacts[k][0]

			admit, err := artInQueue.GetPackage().Admit(art.GetPackage())
			if err != nil || !admit {
				return admit, err
			}
		}
	}

	return true, nil
}

func (s *Solver) artefactIsInConflict(art *artifact.PackageArtifact) bool {
	cc, ok := s.conflictsMap.Packages[art.GetPackage().PackageName()]
	if !ok {
		return false
	}

	for _, c := range cc {
		// Check if propagate error
		valid, _ := c.Admit(art.GetPackage())
		if !valid {
			return true
		}
	}

	return false
}

func (s *Solver) prepareConflictsAndSystemMap(systemPkgs *pkg.Packages) {
	// Prepare the conflicts map to speedup checks
	s.conflictsMap = pkg.NewPkgsMapList()
	// Prepare the system packages map to speedup check
	s.systemMap = pkg.NewPkgsMapList()
	// Prepare the provides map of the installed packages.
	s.providesMap = pkg.NewPkgsMapList()

	for _, p := range *systemPkgs {
		if !s.Opts.IgnoreConflicts {
			pconflicts := p.GetConflicts()
			if len(pconflicts) > 0 {
				for _, c := range pconflicts {
					s.conflictsMap.Add(c.PackageName(), p.(*pkg.DefaultPackage))
				}
			}
		}

		if len(p.(*pkg.DefaultPackage).Provides) > 0 {
			for _, prov := range p.(*pkg.DefaultPackage).Provides {
				s.providesMap.Add(prov.PackageName(), p.(*pkg.DefaultPackage))
			}
		}

		s.systemMap.Add(p.PackageName(), p.(*pkg.DefaultPackage))

	}

}

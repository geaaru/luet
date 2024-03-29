/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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

	Database pkg.PackageDatabase               `yaml:"-" json:"-"`
	Searcher wagon.Searcher                    `yaml:"-" json:"-"`
	MapRepos map[string]*wagon.WagonRepository `yaml:"-" json:"-"`

	conflictsMap     *pkg.PkgsMapList       `yaml:"-" json:"-"`
	systemMap        *pkg.PkgsMapList       `yaml:"-" json:"-"`
	providesMap      *pkg.PkgsMapList       `yaml:"provides,omitempty" json:"provides,omitempty"`
	requiresMap      *pkg.PkgsMapList       `yaml:"-" json:"-"`
	availableArtsMap *artifact.ArtifactsMap `yaml:"-" json:"-"`
	candidatesMap    *artifact.ArtifactsMap `yaml:"-" json:"-"`

	mutex *sync.Mutex `yaml:"-" json:"-'`
}

func NewSolver(cfg *config.LuetConfig, opts *SolverOpts) *Solver {
	return &Solver{
		Config:        cfg,
		Opts:          opts,
		Database:      nil,
		MapRepos:      nil,
		candidatesMap: artifact.NewArtifactsMap(),
		mutex:         &sync.Mutex{},
	}
}

func (s *Solver) SetDatabase(d pkg.PackageDatabase) { s.Database = d }

func (s *Solver) createThinPkgsPlist(p2i *artifact.ArtifactsPack, p2imap *artifact.ArtifactsMap) []*pkg.PackageThin {

	// Instead to check if a dependency is already installed
	// I check if it's present in the map of the packages
	// to install. If isn't present means that is already
	// installed.
	// I follow this choice because excluding the initial
	// installation normally there are less packages to install
	// and more packages already installed.

	// Build the package thin array
	pthinarr := []*pkg.PackageThin{}

	for _, a := range p2i.Artifacts {
		pt := a.GetPackage().ToPackageThin()

		requires := []*pkg.PackageThin{}
		// Check if requires are installed and drop them from the list
		for _, r := range pt.Requires {

			_, present := p2imap.Artifacts[r.PackageName()]
			if present {
				requires = append(requires, r)
			}
		}

		pt.Requires = requires

		pthinarr = append(pthinarr, pt)
	}

	return pthinarr
}

func (s *Solver) recursiveCheckDeps(p *pkg.PackageThin,
	stackRef *[]*pkg.PackageThin, queueRef *map[string]*pkg.PackageThin,
	pinjectRef *map[string]bool) error {

	Debug(fmt.Sprintf("[%s] with stack %s and requires %d", p, *stackRef, len(p.Requires)))
	if pkg.PackageThinIsInList(p, stackRef) {
		Warning(fmt.Sprintf(
			":ambulance: Found deps cycle for package %s: %s. Trying to break it. Please, fix specs or open an issue!!!",
			p.PackageName(), *stackRef))

		p.BreakCyclesOnRequires(stackRef)
	}
	*stackRef = append(*stackRef, p)
	queue := *queueRef
	pinject := *pinjectRef

	for _, r := range p.Requires {
		// The requires of the package doesn't contain the dependencies.
		// I need to use the queue element.

		rq, ok := queue[r.PackageName()]
		if !ok {
			// Check if the dependencies is already injected
			_, injected := pinject[r.PackageName()]

			if !injected {
				return fmt.Errorf("[%s] Error on retrieve requires %s on queue",
					p.HumanReadableString(), r.PackageName())
			}
			// POST: Ignoring this dependency
			Debug(fmt.Sprintf("[%s] Dependency %s already injected. Ignoring.",
				p.HumanReadableString(), r.HumanReadableString()))
			continue
		}

		err := s.recursiveCheckDeps(rq, stackRef, queueRef, pinjectRef)
		if err != nil {
			return err
		}
	}

	return nil

}

func (s *Solver) depCycleDetector(queueRef *map[string]*pkg.PackageThin, pinjectRef *map[string]bool) error {
	elems := []*pkg.PackageThin{}
	queue := *queueRef
	pinject := *pinjectRef

	// Convert map to list
	for _, p := range queue {
		elems = append(elems, p)
	}

	// Sort elements list
	pkg.SortPackageThinList4Requires(&elems)

	nelems := len(elems)
	for idx, p := range elems {
		stack := []*pkg.PackageThin{}
		Debug(fmt.Sprintf(
			"[%d of %d] Checking %s with %d requires...", idx+1, nelems, p.HumanReadableString(),
			len(p.Requires)))

		err := s.recursiveCheckDeps(p, &stack, queueRef, pinjectRef)
		if err != nil {
			return err
		}

		// Check that all dependencies are availables.
		for _, r := range p.Requires {
			_, injected := pinject[r.PackageName()]
			_, inQueue := queue[r.PackageName()]

			if !injected && !inQueue {
				return fmt.Errorf("[%s] Found dependency not resolvable %s",
					p.PackageName(), r.HumanReadableString())
			}
		}
	}

	return nil
}

func (s *Solver) sortPkgsThinArr(refarr *[]*pkg.PackageThin) error {
	ans := []*pkg.PackageThin{}
	pinject := make(map[string]bool, 0)
	queue := make(map[string]*pkg.PackageThin, 0)

	pkg.SortPackageThinList4Requires(refarr)

	start := time.Now()
	for _, p := range *refarr {

		injected := false

		if !p.HasRequires() {
			ans = append(ans, p)
			pinject[p.PackageName()] = true
			injected = true

		} else {
			allReqok := true

			for _, r := range p.Requires {
				if _, ok := pinject[r.PackageName()]; !ok {
					allReqok = false
					break
				}
			}

			if allReqok {
				ans = append(ans, p)
				pinject[p.PackageName()] = true
				injected = true

			} else {
				queue[p.PackageName()] = p
			}

		}

		if injected {
			// POST: check if the elements in queue
			//       could be injected.

			pkgs2remove := []string{}
			for k, pr := range queue {

				allReqok := true
				for _, r := range pr.Requires {
					if _, ok := pinject[r.PackageName()]; !ok {
						allReqok = false
						break
					}
				}

				if allReqok {
					ans = append(ans, pr)
					pinject[pr.PackageName()] = true
					pkgs2remove = append(pkgs2remove, k)
				}

			}

			for _, rm := range pkgs2remove {
				delete(queue, rm)
			}
		}

	} // end for
	Debug(fmt.Sprintf("First sort iteration done in %d µs. Queue size is %d",
		time.Now().Sub(start).Nanoseconds()/1e3, len(queue)))

	if len(queue) > 0 {
		// TODO: review with a more optimized logic

		loopCyclesDetectorCounter := 0
		lastQueueSize := 0
		detectorExecuted := false

		for len(queue) > 0 {
			if len(queue) == lastQueueSize {
				if loopCyclesDetectorCounter > 3 {
					if detectorExecuted {
						return fmt.Errorf(
							"Unexpected error on sort queue of size %d",
							len(queue))
					}
					detectorExecuted = true

					err := s.depCycleDetector(&queue, &pinject)
					if err != nil {
						return err
					}

				} else {
					loopCyclesDetectorCounter++
				}
			} else {
				lastQueueSize = len(queue)
				loopCyclesDetectorCounter = 0
			}

			pkgs2remove := []string{}
			for k, p := range queue {

				allReqok := true
				for _, r := range p.Requires {
					if _, ok := pinject[r.PackageName()]; !ok {
						// Check if there are dependency cycles.
						// I prefer leave this check only here to reduce
						// the impact of a wrong order.
						rr, _ := queue[r.PackageName()]
						if rr != nil && rr.RequirePackage(p) {
							Warning(fmt.Sprintf(
								"Found dependency cycle between %s and %s. I break cycle. Check the repo.",
								p.HumanReadableString(), r.HumanReadableString()))
							allReqok = true
						} else {
							Debug(fmt.Sprintf("[%s] The dependency %s has deps: %s",
								p.HumanReadableString(), r.HumanReadableString(), r.Requires))
							allReqok = false
						}
						break
					} else {
						Debug(fmt.Sprintf("[%s] Dependency %s already injected.", p.HumanReadableString(),
							r.PackageName()))
					}
				}

				if allReqok {
					ans = append(ans, p)
					pinject[p.PackageName()] = true
					pkgs2remove = append(pkgs2remove, k)
				}

			}

			for _, rm := range pkgs2remove {
				delete(queue, rm)
			}

		}
	}

	*refarr = ans

	return nil
}

func (s *Solver) OrderOperations(p2i, p2u, p2r *artifact.ArtifactsPack) (*[]*Operation, error) {
	ans := []*Operation{}
	tmpOps := []*Operation{}

	// PRE: A package could not be available on both p2i and p2u arrays.

	if p2i == nil || p2u == nil || p2r == nil {
		return &ans, errors.New("Invalid parameters to OrderOperations")
	}

	Debug(fmt.Sprintf(
		"Sorting %d install, %d updates, %d removes.",
		len(p2i.Artifacts), len(p2u.Artifacts), len(p2r.Artifacts)))

	// Merge packages to install with packages to updates.
	mergedPack := artifact.NewArtifactsPack()
	mergedPack.Artifacts = p2i.Artifacts
	mergedPack.Artifacts = append(mergedPack.Artifacts, p2u.Artifacts...)
	// Merge map contains all packages to add and/or update
	mergedMap := mergedPack.ToMap()

	if len(mergedPack.Artifacts) == 1 {
		if len(p2i.Artifacts) > 0 {
			tmpOps = append(tmpOps, NewOperation(AddPackage, mergedPack.Artifacts[0]))
		} else {
			tmpOps = append(tmpOps, NewOperation(UpdatePackage, mergedPack.Artifacts[0]))
		}
	} else if len(mergedPack.Artifacts) > 1 {

		start := time.Now()
		pthinarr := s.createThinPkgsPlist(mergedPack, mergedMap)

		Debug(fmt.Sprintf("Creation of %d thin pkgs list in %d µs.", len(pthinarr),
			time.Now().Sub(start).Nanoseconds()/1e3))

		err := s.sortPkgsThinArr(&pthinarr)
		if err != nil {
			return nil, err
		}
		mergedPack = nil

		p2imap := p2i.ToMap()
		p2umap := p2u.ToMap()

		for _, p := range pthinarr {
			var op *Operation = nil
			if _, present := p2imap.Artifacts[p.PackageName()]; present {
				a, _ := p2imap.Artifacts[p.PackageName()]
				op = NewOperation(AddPackage, a[0])
			} else {
				a, _ := p2umap.Artifacts[p.PackageName()]
				op = NewOperation(UpdatePackage, a[0])
			}

			tmpOps = append(tmpOps, op)
		}
		pthinarr = nil

	}

	if len(p2r.Artifacts) > 0 {

		// Sort packages to remove
		p2rmap := p2r.ToMap()
		pthinarr := s.createThinPkgsPlist(p2r, p2rmap)

		if len(tmpOps) == 0 {
			// POST: If there are packages to update/add it means
			//       that there are only remove operations.
			for _, a := range pthinarr {
				val, _ := p2rmap.Artifacts[a.PackageName()]
				ans = append(ans, NewOperation(RemovePackage, val[0]))
			}
		} else {
			// POST: There are both packages to remove and to install/upgrade.

			newRemoves := []*Operation{}

			// Check and Add all packages not available
			// between new install/updates in the right order.
			// The packages to remove are added at the head of the list.
			for _, a := range p2r.Artifacts {
				p := a.GetPackage()
				if _, present := mergedMap.Artifacts[p.PackageName()]; !present {
					newRemoves = append(newRemoves,
						NewOperation(RemovePackage, a),
					)
				}
			}

			if len(newRemoves) > 0 {
				ans = newRemoves
			}

			// Add all packages available on mergedPack in the
			// order of the previous sort.

			idxOps := len(ans)
			rmOps := len(ans)
			tmpNOps := len(tmpOps)
			for tidx := 0; tidx < tmpNOps; tidx++ {
				p := tmpOps[tidx].Artifact.GetPackage()

				// Check if the operation has a remove operation
				if pr, present := p2rmap.Artifacts[p.PackageName()]; present {

					idxConflict := -1
					for idx := idxOps - 1; idx > rmOps; idx-- {
						pp := ans[idx].Artifact.GetPackage()
						admit, err := pp.Admit(p)
						if err != nil {
							Warning(fmt.Sprintf("[%s] Error on check conflict with %s: %s",
								p.PackageName(), pp.PackageName(), err.Error()))
						} else {
							if !admit {
								Debug(fmt.Sprintf(
									"[%s] Conflicts with %s (at pos %d).",
									p.PackageName(), pp.HumanReadableString(), idx))
								idxConflict = idx
								// We consider the last conflicts the point
								// where add the uninstall of the package before
							}
						}
					}

					if idxConflict == 0 {
						ans = append([]*Operation{
							NewOperation(RemovePackage, pr[0]),
						}, ans...)
						ans = append(ans, tmpOps[tidx])
						idxOps++
					} else if idxConflict < 0 {
						ans = append(ans,
							[]*Operation{
								NewOperation(RemovePackage, pr[0]),
								tmpOps[tidx],
							}...)
						idxOps++
					} else {

						// NOTE: Using slice and modify the linked array modify the slice too.
						//       So, i using tmpAns and copy every operation with a for. maybe we could find
						//       a better solution?
						segment := ans[idxConflict:]

						tmpAns := make([]*Operation, idxConflict)
						for idx := 0; idx < idxConflict; idx++ {
							tmpAns[idx] = ans[idx]
						}
						tmpAns = append(tmpAns, NewOperation(RemovePackage, pr[0]))

						// I need to add 1 because this is a pointer to the
						// existing array.
						tmpAns = append(tmpAns, segment...)
						tmpAns = append(tmpAns, tmpOps[tidx])
						ans = tmpAns
						idxOps++
					}
				} else {
					ans = append(ans, tmpOps[tidx])
				}
				idxOps++
			}

		}

		pthinarr = nil
		p2rmap = nil
	} else {
		ans = tmpOps
	}

	mergedMap = nil

	return &ans, nil
}

func (s *Solver) Install(pkgsref *[]*pkg.DefaultPackage) (*artifact.ArtifactsPack, *artifact.ArtifactsPack, error) {
	ans2Install := artifact.NewArtifactsPack()
	ans2Remove := artifact.NewArtifactsPack()

	if s.Database == nil {
		return nil, nil, errors.New("Solver Install requires Database")
	}

	pkgs := *pkgsref
	// PRE: the input packages are with valid category/name strings.

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
		IgnoreMasks:      s.Opts.IgnoreMasks,
	}

	err := s.prepareSearcher()
	if err != nil {
		return nil, nil, err
	}

	// For every package in list retrieve all available candidates
	// and store the result on ArtifactsMap
	reposArtifacts, err := s.Searcher.SearchArtifacts(searchOpts)
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

	s.prepareConflictsAndSystemMap(&systemPkgs, false)

	// Create the repositories map
	s.MapRepos = make(map[string]*wagon.WagonRepository, 0)
	for idx, repo := range s.Config.SystemRepositories {
		if !repo.Enable {
			continue
		}

		repobasedir := s.Config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
		wr := wagon.NewWagonRepository(&s.Config.SystemRepositories[idx])
		err := wr.ReadWagonIdentify(repobasedir)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"Error on read repository identity file: " + err.Error(),
			)
		}

		s.MapRepos[repo.Name] = wr
	}

	// NOTE: Using a common search for all packages doesn't
	//       permit to rerieve the map between package used on
	//       search and the packages returned when it's used
	//       the provides.
	//       To resolve this I need to iterate over all packages
	//       and to create a provides map to join common packages.
	provMaps := make(map[string]map[string][]*artifact.PackageArtifact, 0)
	for _, arts := range s.availableArtsMap.Artifacts {
		// NOTE: I need to iterate over all value packages because provides
		//       could be different.
		ppmap := make(map[string]bool, 0)
		for _, art := range arts {
			p := art.GetPackage()
			if p.HasProvides() {
				for _, prov := range p.GetProvides() {
					ppmap[prov.PackageName()] = true
				}
			}
		} // end for arts

		if len(ppmap) > 0 {
			p := arts[0].GetPackage()
			for k, _ := range ppmap {
				if val, ok := provMaps[k]; ok {
					if pval, ok2 := val[p.PackageName()]; ok2 {
						val[p.PackageName()] = append(pval, arts...)
					} else {
						val[p.PackageName()] = arts
					}
					provMaps[k] = val
				} else {
					val := make(map[string][]*artifact.PackageArtifact, 0)
					val[p.PackageName()] = arts
					provMaps[k] = val
				}
			}
		}

		ppmap = nil
	}
	// Check generated prov map
	if len(provMaps) > 0 {
		for k, m := range provMaps {
			if len(m) > 1 {
				arts := []*artifact.PackageArtifact{}
				installedPkg := ""
				for k, pp := range m {
					if _, present := s.systemMap.Packages[k]; present {
						installedPkg = k
						break
					}
					arts = append(arts, pp...)
				}

				if installedPkg != "" {

					Debug(fmt.Sprintf(
						"For the provides %s found multiple packages. But is already installed %s.",
						k, installedPkg))

					// Delete all packages because is already installed.
					for k, _ := range m {
						delete(s.availableArtsMap.Artifacts, k)
					}
				} else {
					// Sort packages for requires and repos
					wagon.SortArtifactList4ReposAndRequires(
						&arts, &s.MapRepos)

					Debug(fmt.Sprintf(
						"For provide %s found multiple packages. Using %s",
						k, arts[0].GetPackage().PackageName()))

					// Delete the packages related with the same provides
					// loser.
					for k, _ := range m {
						if k != arts[0].GetPackage().PackageName() {
							delete(s.availableArtsMap.Artifacts, k)
						}
					}

				}

			}
		}
	}

	// Process all selected packages to install.
	// Created the key list to permit changes on the map
	// meantime that packages are elaborated.
	pList := []string{}
	for pname, _ := range s.availableArtsMap.Artifacts {
		pList = append(pList, pname)
	}

	for _, pname := range pList {
		err := s.resolvePackage(pname, []string{})
		if !s.Opts.Force && err != nil {
			return nil, nil, err
		}
	}

	// Cleanup resources
	s.systemMap = nil
	s.conflictsMap = nil
	s.providesMap = nil
	// Cleanup memory
	s.MapRepos = nil

	// Create the list of package to install
	if s.Opts.NoDeps {
		for _, pkg := range pkgs {
			plist, _ := s.candidatesMap.Artifacts[pkg.PackageName()]
			if len(plist) == 0 {
				// POST: Check if the package is provided
				plist = s.candidatesMap.GetProvides(pkg.PackageName())
				if len(plist) == 0 {
					Warning(fmt.Sprintf("No candidates found for %s", pkg.PackageName()))
				} else {
					ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
				}
			} else {
				ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
			}
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
	//  means that there aren't packages that requires it.

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
			Debug(fmt.Sprintf(
				"For %s not found admitted dependencies.",
				art.GetPackage().HumanReadableString()))
			continue
		}

		foundMatched = true
		break
	}

	if !foundMatched {
		var str string
		if len(stack) > 0 {
			str = fmt.Sprintf(
				"No valid or admitted version found for dependency %s", pkgstr)
		} else {
			str = fmt.Sprintf(
				"No valid candidate or valid dependencies found for %s", pkgstr)
		}
		Debug(str)
		return errors.New(str)
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
			// Check if the dependency installed is admitted by the
			// package to install
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

		// POST: Check if the dependency is in the list of the packages
		//       to elaborate and to install
		var reposArtifacts *[]*artifact.PackageArtifact
		var err error
		provStr := ""

		if _, onQueue := s.availableArtsMap.Artifacts[p.PackageName()]; onQueue {
			// Sort all available versions of the selected package.
			// The first is the major version
			if s.MapRepos != nil {
				selectedArts, err := s.availableArtsMap.GetArtifactsByKey(p.PackageName())
				if err != nil {
					return false, err
				}

				wagon.SortArtifactList4VersionAndRepos(&selectedArts,
					&s.MapRepos, true)

				reposArtifacts = &selectedArts
			} else {
				selectedArts, err := s.availableArtsMap.GetSortedArtifactsByKey(p.PackageName())
				if err != nil {
					return false, err
				}
				reposArtifacts = &selectedArts
			}

		} else {

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
				IgnoreMasks:      s.Opts.IgnoreMasks,
			}
			Debug(fmt.Sprintf("[%30s] Searching for dependency %s...",
				candidate.PackageName(), searchOpts.Packages[0].PackageName()))
			reposArtifacts, err = s.Searcher.SearchArtifacts(searchOpts)
			if err != nil {
				return false, err
			}

			if s.MapRepos != nil {
				// Sort packages for version and repos (on reverse)
				wagon.SortArtifactList4VersionAndRepos(
					reposArtifacts, &s.MapRepos, true)
			}

			// Convert the results in a map with all available versions of the
			// selected packages.
			for _, depArt := range *reposArtifacts {
				// If the researched package is provided we need to use
				// the name of the package that provides the requirements.
				if provStr == "" &&
					depArt.GetPackage().PackageName() != searchOpts.Packages[0].PackageName() {
					provStr = depArt.GetPackage().PackageName()
				}

				s.availableArtsMap.Add(depArt)
			}

		}

		if provStr != "" {
			err = s.resolvePackage(provStr, stack)
		} else {
			err = s.resolvePackage(p.PackageName(), stack)
		}
		if err != nil {
			if strings.HasPrefix(err.Error(), "No valid") {
				return false, nil
			}
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
			if err != nil {
				return admit, err
			} else if !admit {
				Debug(fmt.Sprintf("%s NOT admits %s...",
					artInQueue.GetPackage().HumanReadableString(),
					art.GetPackage().HumanReadableString()))
				return admit, err
			}
		}
	}

	return true, nil
}

func (s *Solver) artefactIsInConflict(art *artifact.PackageArtifact) bool {
	p := art.GetPackage()
	cc, ok := s.conflictsMap.Packages[p.PackageName()]
	if ok {
		for _, c := range cc {
			// Check if propagate error
			valid, _ := c.Admit(p)
			if !valid {
				return true
			}
		}
	}

	// Check if the artefact has conflicts with existing tree.
	for _, c := range p.PackageConflicts {
		val, present := s.systemMap.Packages[c.PackageName()]
		if present {
			if valid, _ := p.Admit(val[0]); !valid {
				// Check if the package will replace this.
				prov := p.GetProvidePackage(val[0].PackageName())
				if prov != nil {
					Debug(fmt.Sprintf(
						"[%s] conflict with %s but is provided. Ignoring it.",
						p.HumanReadableString(), val[0].HumanReadableString()))
				} else {
					return true
				}
			}
		}
	}

	return false
}

func (s *Solver) prepareConflictsAndSystemMap(systemPkgs *pkg.Packages, withReverseRequires bool) {
	// Prepare the conflicts map to speedup checks
	s.conflictsMap = pkg.NewPkgsMapList()
	// Prepare the system packages map to speedup check
	s.systemMap = pkg.NewPkgsMapList()
	// Prepare the provides map of the installed packages.
	s.providesMap = pkg.NewPkgsMapList()

	if withReverseRequires {
		s.requiresMap = pkg.NewPkgsMapList()
	}

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

		if withReverseRequires {
			// POST: Register the map of the reverse requires for all
			//       required packages of the selected package.
			if len(p.(*pkg.DefaultPackage).PackageRequires) > 0 {
				for _, req := range p.(*pkg.DefaultPackage).PackageRequires {
					s.requiresMap.Add(req.PackageName(), p.(*pkg.DefaultPackage))
				}
			}
		}

		s.systemMap.Add(p.PackageName(), p.(*pkg.DefaultPackage))
	}

}

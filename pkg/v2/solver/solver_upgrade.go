/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	"github.com/geaaru/luet/pkg/v2/repository/mask"
	"golang.org/x/sync/semaphore"
)

func (s *Solver) Upgrade() (*artifact.ArtifactsPack, *artifact.ArtifactsPack, *artifact.ArtifactsPack, error) {

	ans2Install := artifact.NewArtifactsPack()
	ans2Remove := artifact.NewArtifactsPack()
	ans2Update := artifact.NewArtifactsPack()

	if s.Database == nil {
		return nil, nil, nil, errors.New("Solver Install requires Database")
	}

	// TODO: Use a different solution with less memory usage
	systemPkgs := s.Database.World()

	s.prepareConflictsAndSystemMap(&systemPkgs, true)

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
			return nil, nil, nil, fmt.Errorf(
				"Error on read repository identity file: " + err.Error(),
			)
		}

		s.MapRepos[repo.Name] = wr
	}

	Debug(":brain:Starting preliminary analysis...")
	start := time.Now()
	// 1. Search for all packages with new versions excluding
	//    his dependencies or with changes on requires/conflicts/provides
	err := s.analyzeInstalledPackages()
	if err != nil {
		return nil, nil, nil, err
	}
	Debug(fmt.Sprintf(":brain:Preliminary analysis done in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	if len(s.availableArtsMap.Artifacts) == 0 {
		// POST: No updates available
		return ans2Remove, ans2Update, ans2Install, nil
	}

	// temporary convert the map to list for the sort.
	artsSortList := s.availableArtsMap.ToList()

	// Sort packages for requires and repos
	wagon.SortArtifactList4RequiresAndRepos(
		artsSortList, &s.MapRepos)

	// POST: There are packages to upgrade or possible upgrades.
	//       Analyze every candidate.
	pList := []string{}
	mapArt := make(map[string]bool, 0)
	for _, art := range *artsSortList {
		pname := art.GetPackage().PackageName()
		if _, present := s.availableArtsMap.Artifacts[pname]; !present {
			// POST: Package provides an installed package.
			key := s.availableArtsMap.GetKeyFromValue(art)
			if key == "" {
				Warning(fmt.Sprintf(
					"No key found for package %s",
					art.GetPackage().HumanReadableString()))
				continue
			}
			pname = key
		}
		if _, ok := mapArt[pname]; ok {
			continue
		}
		mapArt[pname] = true
		pList = append(pList, pname)
	}
	mapArt = nil
	artsSortList = nil

	for _, pname := range pList {
		err := s.checkCandidate2Upgrade(pname, []string{})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Cleanup memory
	s.MapRepos = nil

	// Create the list of the packages to install, upgrade,
	// and remove.
	if s.Opts.NoDeps {
		for _, pname := range pList {
			plist, _ := s.candidatesMap.Artifacts[pname]

			val, ok := s.systemMap.Packages[pname]
			if ok {
				// val is a DefaultPackage that I add to an empty
				// artefact with only Runtime attribute.
				// For remove is needed only to create a Stone object.
				ans2Remove.Artifacts = append(ans2Remove.Artifacts,
					&artifact.PackageArtifact{
						Runtime: val[0],
					})

				if plist[0].GetPackage().PackageName() != pname {
					// The provides replace the exiting package too.
					// I add it if there is also the same package as update.
					if _, present := s.candidatesMap.Artifacts[plist[0].GetPackage().PackageName()]; !present {
						ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
					} // else ignoring it.
				} else {
					ans2Update.Artifacts = append(ans2Update.Artifacts, plist[0])
				}

				// POST: Package to upgrade
			} else {
				// POST: New package.
				ans2Install.Artifacts = append(ans2Install.Artifacts, plist[0])
			}

		}
	} else {
		for pname, _ := range s.candidatesMap.Artifacts {
			plist, _ := s.candidatesMap.Artifacts[pname]
			acandidate := plist[0]

			val, ok := s.systemMap.Packages[pname]
			if ok {
				// POST: Package to upgrade

				// val is a DefaultPackage that I add to an empty
				// artefact with only Runtime attribute.
				// For remove is needed only to create a Stone object.
				ans2Remove.Artifacts = append(ans2Remove.Artifacts,
					&artifact.PackageArtifact{
						Runtime: val[0],
					})

				if acandidate.GetPackage().PackageName() != pname {
					// The provides replace the exiting package too.
					// I add it if there is also the same package as update.
					if _, present := s.candidatesMap.Artifacts[acandidate.GetPackage().PackageName()]; !present {
						ans2Install.Artifacts = append(ans2Install.Artifacts, acandidate)
					} // else ignoring it.
				} else {
					ans2Update.Artifacts = append(ans2Update.Artifacts, acandidate)
				}

				// Check if the package provides packages installed
				if acandidate.GetPackage().HasProvides() {
					for _, prov := range acandidate.GetPackage().GetProvides() {
						if pr, present := s.systemMap.Packages[prov.PackageName()]; present {
							Debug(fmt.Sprintf("[%s] provides and replace the existing %s.",
								acandidate.GetPackage().PackageName(),
								pr[0].HumanReadableString()))

							art2rm := &artifact.PackageArtifact{
								Runtime: pr[0],
							}

							if !ans2Remove.IsPresent(art2rm) {
								ans2Remove.Artifacts = append(ans2Remove.Artifacts, art2rm)
							}
						}
					}
				}

			} else {
				// POST: New package.

				// Check if the package provides packages installed.
				// This could happens for example when a new package is available
				// and this replace an old package with a different slot.
				// For example: sys-libs/libunwind:7 (funtoo 1.4) to sys-libs/libunwind.
				// NOTE: In this case the sys-libs/libunwind is injected as candidate
				//       because is a dependency of another installed package and so,
				//       the upgrade process doesn't see it as an upgrade.
				if acandidate.GetPackage().HasProvides() {
					for _, prov := range acandidate.GetPackage().GetProvides() {
						if pr, present := s.systemMap.Packages[prov.PackageName()]; present {
							Debug(fmt.Sprintf("[%s] provides and replace the existing %s.",
								acandidate.GetPackage().PackageName(),
								pr[0].HumanReadableString()))

							art2rm := &artifact.PackageArtifact{
								Runtime: pr[0],
							}

							if !ans2Remove.IsPresent(art2rm) {
								ans2Remove.Artifacts = append(ans2Remove.Artifacts, art2rm)
							}
						}
					}
				}

				ans2Install.Artifacts = append(ans2Install.Artifacts, acandidate)
			}

		}

	}

	// NOTE: At the moment the dependencies to remove for conflicts
	//       from existing rootfs are not yet supported
	//       automatically.

	return ans2Remove, ans2Update, ans2Install, nil
}

// This function does something more similar to resolvePackage but
// consider that the selected package is available in the system
// It could be possible that in the next refactor will be joined
// with the resolvePackage function.
func (s *Solver) checkCandidate2Upgrade(pkgstr string, stack []string) error {
	// Breaks check cycles
	if helpers.ContainsElem(&stack, pkgstr) {
		// POST: this dependency/package is already been elaborated.
		return nil
	}

	// NOTE: Hereinafter, a summary of the checks needed:
	// 1. Retrieve the version with the major version that is admitted
	//    by all installed packages.
	// 1b. Update the package with the same version but with different
	//     hash.
	// 2. Check dependencies of the selected package.
	//    If the dependency is been selected for the upgrade, validate
	//    the new before the rest.
	// 3. Check that all packages that requires the candidates
	//    admit the updates.
	//
	// The candidates are order for requires and repository priority.
	// The first wins respect the next for the upgrade.

	_, ok := s.candidatesMap.Artifacts[pkgstr]
	if ok {
		// POST: the package is already been elaborated
		return nil
	}

	var selectedArts []*artifact.PackageArtifact
	var err error

	// If the package replace an existing package through provides
	// could not be present in the availableArtsMap with his
	// package name. If doesn't exist search for the provided package.

	// Sort all available versions of the selected package.
	// The first is the major version
	if s.MapRepos != nil {
		selectedArts, err = s.availableArtsMap.GetArtifactsByKey(pkgstr)
		if err != nil {
			return err
		}
		wagon.SortArtifactList4VersionAndRepos(&selectedArts,
			&s.MapRepos, true)
	} else {
		selectedArts, err = s.availableArtsMap.GetSortedArtifactsByKey(pkgstr)
		if err != nil {
			return err
		}
	}
	//pkg2replace = pkgstr

	// Retrieve the DefaultPackage of the installed package.
	dp, ok := s.systemMap.Packages[pkgstr]
	if !ok {
		return fmt.Errorf("Unexpected condition on retrieve package for %s",
			pkgstr)
	}

	gpI, _ := dp[0].ToGentooPackage()
	pHash := dp[0].GetComparitionHash()

	foundMatched := false
	foundEqual := false
	// The following code is pretty similar to initial upgrades check
	// but with the mission to validate the result to all other installed
	// packages.

	// Map to avoid processing of the same version multiple time.
	// This is possible when the same version of the package is
	// available multiple time in different repositories.
	bannedVersion := make(map[string]bool, 0)

	for idx, _ := range selectedArts {
		art := selectedArts[idx]
		candidate := art.GetPackage()

		if !candidate.AtomMatches(dp[0]) {
			// POST: The package provides the selected package to analyze.
			prov := candidate.GetProvidePackage(dp[0].PackageName())
			if prov == nil {
				Warning(fmt.Sprintf("Unexpected package %s for installed package %s",
					candidate.HumanReadableString(), dp[0].PackageName()))
				continue
			}
			candidate = prov
		}

		gpS, err := candidate.ToGentooPackage()
		if err != nil {
			return err
		}

		// If version is already been processed and banned I will
		// skip the artefact
		if _, ok := bannedVersion[candidate.GetVersion()]; ok {
			continue
		}

		val, err := gpS.GreaterThan(gpI)
		if err != nil {
			return err
		}

		if val {
			// POST: There is a new version with a value
			//       greather then the installed version.

			// Check if the selected version is valid for
			// the installed packages with a requires with
			// the selected package and for the candidates.

			users, ok := s.requiresMap.Packages[candidate.PackageName()]
			if ok {
				usersAdmitNew := true
				// Check if the new packages is admitted by the existing users
				for _, user := range users {
					admit, _ := user.Admit(candidate)
					if !admit {
						// POST: the candidate is not admitted from existing
						//       package. Before block the candidate I check if there is a new
						//       version for the user package that admit the new
						//       version that will be analyzed later.
						newUserVersions, ok := s.availableArtsMap.Artifacts[user.PackageName()]
						if ok {
							newUserVersionAdmit := false
							// TODO: Handle provides
							for _, uv := range newUserVersions {
								a, err := uv.GetPackage().Admit(candidate)
								if err != nil {
									return err
								}
								if a {
									newUserVersionAdmit = true
								}
							}

							if !newUserVersionAdmit {
								usersAdmitNew = false
								break
							}

						} else {
							usersAdmitNew = false
							break
						}
					}
				} // end for users

				if !usersAdmitNew {
					continue
				}

			} else {
				// POST: No packages that requires the selected package.
				//       Good for the next checks.
			}

			// Check if the selected package is in conflicts with
			// existing tree.
			// The conflicts related to the new packages are checked by the
			// artefactAdmitByQueue in order of processing order and
			// by the Admit method.
			if !s.Opts.IgnoreConflicts && s.artefactIsInConflict(art) {
				bannedVersion[candidate.GetVersion()] = true
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
			admittedDeps, err := s.processArtefactDeps4Upgrade(art, ss)
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

		} else if val, _ := gpS.Equal(gpI); val {

			// NOTE: if the version is the same
			foundEqual = true

			aHash := candidate.GetComparitionHash()

			if pHash != aHash {
				// POST: There aren't new release with a version
				//       greather then the installed. I could just
				//       validate the new requires.

				ss := append(stack, art.GetPackage().PackageName())
				// Check the package dependencies
				admittedDeps, err := s.processArtefactDeps4Upgrade(art, ss)
				if err != nil {
					return err
				}

				if !admittedDeps {
					bannedVersion[candidate.GetVersion()] = true
					continue
				}

				foundMatched = true
				break

			} else {
				bannedVersion[candidate.GetVersion()] = true
				// POST: Nothing to do.
				// TODO: Maybe i need to elaborated yet things to get
				//       a version that is valid with the conflicts
				//       of the packages installed and/or to install.
				break
			}

		} else if s.Opts.Deep && !foundEqual {
			// POST: the candidate is good to downgrade of the exiting
			//       package

			// Check if the selected package is in conflicts with
			// existing tree.
			// The conflicts related to the new packages are checked by the
			// artefactAdmitByQueue in order of processing order and
			// by the Admit method.
			if !s.Opts.IgnoreConflicts && s.artefactIsInConflict(art) {
				bannedVersion[candidate.GetVersion()] = true
				continue
			}

			// Validate the selected package with new packages
			// in queue.
			admit, err := s.artefactAdmitByQueue(art)
			if err != nil {
				return err
			}

			if !admit {
				bannedVersion[candidate.GetVersion()] = true
				continue
			}

			ss := append(stack, art.GetPackage().PackageName())
			// Check and in queue all package dependencies
			admittedDeps, err := s.processArtefactDeps4Upgrade(art, ss)
			if err != nil {
				return err
			}

			if !admittedDeps {
				// POST: Not all packages dependencies are admit by
				//       the current system.
				bannedVersion[candidate.GetVersion()] = true
				continue
			}

			foundMatched = true
			break

		} else {
			// TODO: Maybe i need to elaborated yet things to get
			//       a version that is valid with the conflicts
			//       of the packages installed and/or to install.
			bannedVersion[candidate.GetVersion()] = true
		}

	} // end for selectedArts

	if foundMatched {
		// TODO: Check if just needed leave the valid candidate.

		firstValid := false
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

	} // else ignoring package to update.

	return nil
}

func (s *Solver) processArtefactDeps4Upgrade(art *artifact.PackageArtifact, stack []string) (bool, error) {
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

				// POST: the candidate is not admitted from existing installed
				//       packages. Before block the candidate I check if there is a new
				//       version for the user package that admit the new
				//       version that will be analyzed later.
				newDepsVersions, ok := s.availableArtsMap.Artifacts[p.PackageName()]
				if ok {
					admit = false
					for _, dv := range newDepsVersions {
						a, _ := dv.GetPackage().Admit(candidate)
						if a {
							admit = true
							break
						}
					}
				}

				if !admit {
					return false, nil
				}
			}

			// Nothing to do. The dependency is already on system and is valid
			// or there is a new update valid.
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
		// or is in the list of the packages to upgrade
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
			IgnoreMasks:      s.Opts.IgnoreMasks,
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

func (s *Solver) checkInstalledPackageWrapper(
	p *pkg.DefaultPackage, channel chan helpers.ChannelError,
	sem *semaphore.Weighted, waitGroup *sync.WaitGroup,
	ctx *context.Context) {

	defer waitGroup.Done()
	err := sem.Acquire(*ctx, 1)
	if err != nil {
		Error("Error on acquire semaphore: " + err.Error())
		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}
	defer sem.Release(1)

	start := time.Now()
	err = s.checkInstalledPackage(p)
	Debug(fmt.Sprintf(":brain: Analysis %s done in %d µs.",
		p.PackageName(),
		time.Now().Sub(start).Nanoseconds()/1e3))
	if err != nil {
		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}

	channel <- helpers.ChannelError{
		Error:   nil,
		Closure: p,
	}
}

func (s *Solver) checkInstalledPackage(p *pkg.DefaultPackage) error {

	// Clone the installed package to set version >=0
	ps := p.Clone().(*pkg.DefaultPackage)
	ps.Version = ">=0"

	searchOpts := &wagon.StonesSearchOpts{
		Packages:      pkg.DefaultPackages([]*pkg.DefaultPackage{ps}),
		Categories:    []string{},
		Labels:        []string{},
		LabelsMatches: []string{},
		Matches:       []string{},
		FilesOwner:    []string{},
		Annotations:   []string{},
		// Set always hidden because the package could be hidden
		Hidden:           true,
		AndCondition:     false,
		WithFiles:        false,
		WithRootfsPrefix: false,
		Full:             true,
		OnlyPackages:     true,
		IgnoreMasks:      s.Opts.IgnoreMasks,
	}

	start := time.Now()

	// Retrieve all new candidates from repositories.
	reposArtifacts, err := s.Searcher.SearchArtifacts(searchOpts)
	if err != nil {
		return err
	}
	ps = nil

	Debug(fmt.Sprintf(":brain:Search %s done in %d µs (found %d candidates).",
		p.PackageName(),
		time.Now().Sub(start).Nanoseconds()/1e3,
		len(*reposArtifacts)))
	if len(*reposArtifacts) == 0 {
		Debug(fmt.Sprintf(
			"[%s] No artifacts found between repositories. Noop.",
			p.PackageName()))

		// The aren't availables packages about the
		// checked installed. I leave the current package.
		return nil
	}

	// Check if exists a version greather then
	// the installed version. For now I ignore
	// eventually conflicts.
	gpI, err := p.ToGentooPackage()
	if err != nil {
		return err
	}

	pHash := p.GetComparitionHash()

	// Sort packages for version and repos (on reverse)
	wagon.SortArtifactList4VersionAndRepos(
		reposArtifacts, &s.MapRepos, true)

	// To avoid continue reinstall of the same package version
	// with different hash on different repository. For a specific
	// version I consider only the first.
	elabVersion := make(map[string]bool, 0)

	foundNewVersion := false
	// Store candidate name to print the name of the package
	// that provides the searched package.
	candidateName := ""
	for idx, a := range *reposArtifacts {

		provides := false
		ap := a.GetPackage()
		if !ap.AtomMatches(p) {
			// POST: the package provides the package in analysis.
			//       Retrieve the provide package.
			prov := ap.GetProvidePackage(p.PackageName())
			if prov == nil {
				return fmt.Errorf("For package %s found artefact %s but without provides.",
					p.HumanReadableString(), ap.HumanReadableString())
			}
			ap = prov
			provides = true
		}

		if _, present := elabVersion[ap.GetVersion()]; present {
			// If the version is already been elaborated I ignoring
			// the same package version available in the other repositories.
			continue
		}
		elabVersion[ap.GetVersion()] = true
		gpR, err := ap.ToGentooPackage()
		if err != nil {
			return err
		}

		val, err := gpR.GreaterThan(gpI)

		if err != nil {
			return err
		}
		if val {
			// POST: There is at least one version
			//       new. This means that I will elaborate
			//       a specific analysis later.

			if provides {
				Debug(fmt.Sprintf(
					":brain:Provide %s greather than %s.",
					gpR.GetPF(), gpI.GetPF()))
			} else {
				Debug(fmt.Sprintf(
					":brain:Package %s greather than %s.",
					gpR.GetPF(), gpI.GetPF()))
			}
			candidateName = a.GetPackage().PackageName()
			foundNewVersion = true
			break
		} else if val, _ = gpR.Equal(gpI); val {
			// POST: Check if the packages hashes are the same.

			aHash := a.GetPackage().GetComparitionHash()

			if pHash != aHash {

				if provides {
					Debug(fmt.Sprintf(
						":brain:Provide %s has a new hash.",
						gpI.GetPF()))
				} else {
					Debug(fmt.Sprintf(
						":brain:Package %s has a new hash.",
						gpI.GetPF()))
				}
				candidateName = a.GetPackage().PackageName()
				foundNewVersion = true
				break
			}
		} else if s.Opts.Deep && idx == 0 {
			// Checking only if idx == 0 because else
			// means that there is an equal release and I
			// don't need to downgrade.

			if val, _ = gpR.LessThan(gpI); val {
				// POST: There aren't version greather or equal then
				//       the installed version but the deep
				//       option is enabled and I'm searching for
				//       the package to downgrade.
				if provides {
					Debug(fmt.Sprintf(
						":brain:Provide %s less than %s.",
						gpR.GetPFB(), gpI.GetPFB()))
				} else {
					Debug(fmt.Sprintf(
						":brain:Package %s less than %s.",
						gpR.GetPFB(), gpI.GetPFB()))
				}
				candidateName = a.GetPackage().PackageName()
				foundNewVersion = true
				break
			}
		}
	}

	if foundNewVersion {
		s.mutex.Lock()
		Debug(fmt.Sprintf(
			":brain:Package %s queued for analysis.",
			candidateName))
		// POST: Add the versions on availableArtsMap
		s.availableArtsMap.Artifacts[p.PackageName()] = *reposArtifacts
		s.mutex.Unlock()
	}

	return nil
}

func (s *Solver) prepareSearcher() error {
	if s.Searcher == nil {
		s.Searcher = wagon.NewSearcherSimple(s.Config)
		if !s.Opts.IgnoreMasks {
			maskManager := mask.NewPackagesMaskManager(s.Config)
			err := maskManager.LoadFiles()
			if err != nil {
				return err
			}
			s.Searcher.SetMaskManager(maskManager)
		}
	}
	return nil
}

func (s *Solver) analyzeInstalledPackages() error {
	s.availableArtsMap = artifact.NewArtifactsMap()

	err := s.prepareSearcher()
	if err != nil {
		return err
	}

	waitGroup := &sync.WaitGroup{}
	sem := semaphore.NewWeighted(int64(
		s.Config.GetGeneral().Concurrency))
	ctx := context.TODO()

	defer waitGroup.Wait()

	var ch chan helpers.ChannelError = make(
		chan helpers.ChannelError,
		s.Config.GetGeneral().Concurrency,
	)

	nPkgs := 0
	// For every package I check if in the pulled
	// repositories there are new versions or changes
	// and I add the package for a second phase analysis
	// in that case.
	for _, p := range s.systemMap.Packages {
		Debug(fmt.Sprintf(":brain:Checking package %s", p[0].HumanReadableString()))
		waitGroup.Add(1)
		go s.checkInstalledPackageWrapper(
			p[0], ch, sem, waitGroup, &ctx)
		nPkgs++
	}

	res := 0
	if nPkgs > 0 {
		for i := 0; i < nPkgs; i++ {
			resp := <-ch
			if resp.Error != nil {
				res = 1
				p := resp.Closure.(*pkg.DefaultPackage)
				Error(fmt.Sprintf(
					"On analyze package %s: %s",
					p.PackageName(), resp.Error.Error()))
			}
		}
	}

	if res > 0 {
		return errors.New(
			"Unexpected error catched on packages analysis.")
	}

	return nil
}

// Copyright © 2021 Daniele Rondina <geaaru@sabayonlinux.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package solver

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/crillab/gophersat/bf"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
)

const (
	SolverV2ResolverType = "solverv2"
)

type SolverV2 struct {
	*Solver
}

func NewSolverV2(t Options,
	installed, definitiondb, solverdb pkg.PackageDatabase,
	re PackageResolver) *SolverV2 {

	return &SolverV2{
		Solver: &Solver{
			InstalledDatabase:  installed,
			DefinitionDatabase: definitiondb,
			SolverDatabase:     solverdb,
			Resolver:           re,
		},
	}
}

type FormulasGate struct {
	Formulas []bf.Formula
}

func (s *SolverV2) buildFormula(pack pkg.Package, v *map[string]interface{}, f *FormulasGate) error {
	visited := *v

	if _, ok := visited[pack.HumanReadableString()]; ok {
		return nil
	}
	visited[pack.HumanReadableString()] = true
	encodedA, err := s.SolverDatabase.CreatePackage(pack)
	if err != nil {
		return err
	}

	A := bf.Var(encodedA)

	// Do conflict with other packages versions (if A is selected,
	// then conflict with other versions of A)
	packages, _ := s.DefinitionDatabase.FindPackageVersions(pack)
	if len(packages) > 1 {
		for _, cp := range packages {
			if !pack.Matches(cp) {
				encodedB, err := s.SolverDatabase.CreatePackage(cp)
				if err != nil {
					return err
				}
				B := bf.Var(encodedB)
				f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), bf.Or(bf.Not(A), bf.Not(B))))
			}
		}
	}

	for _, requiredDef := range pack.GetRequires() {
		required, err := s.DefinitionDatabase.FindPackage(requiredDef)
		if err != nil || requiredDef.IsSelector() {
			if err == nil {
				requiredDef = required.(*pkg.DefaultPackage)
			}

			packages, err := s.DefinitionDatabase.FindPackages(requiredDef)
			if err != nil || len(packages) == 0 {
				required = requiredDef
			} else {

				var ALO []bf.Formula // , priorityConstraints, priorityALO []bf.Formula
				// AMO/ALO - At most/least one
				for _, o := range packages {
					encodedB, err := s.SolverDatabase.CreatePackage(o)
					if err != nil {
						return err
					}
					B := bf.Var(encodedB)
					ALO = append(ALO, B)
					for _, i := range packages {
						if !o.Matches(i) {
							encodedI, err := s.SolverDatabase.CreatePackage(i)
							if err != nil {
								return err
							}
							I := bf.Var(encodedI)
							f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), bf.Or(bf.Not(I), bf.Not(B))))
						}
					}
				}
				f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), bf.Or(ALO...))) // ALO - At least one
				continue
			}

		}

		encodedB, err := s.SolverDatabase.CreatePackage(required)
		if err != nil {
			return err
		}
		B := bf.Var(encodedB)
		f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), B))
		r := required.(*pkg.DefaultPackage) // We know since the implementation is DefaultPackage, that can be only DefaultPackage
		err = s.buildFormula(r, v, f)
		if err != nil {
			return err
		}

	}

	for _, requiredDef := range pack.GetConflicts() {
		required, err := s.DefinitionDatabase.FindPackage(requiredDef)
		if err != nil || requiredDef.IsSelector() {
			if err == nil {
				requiredDef = required.(*pkg.DefaultPackage)
			}
			packages, err := s.DefinitionDatabase.FindPackages(requiredDef)
			if err != nil || len(packages) == 0 {
				required = requiredDef
			} else {
				if len(packages) == 1 {
					required = packages[0]
				} else {
					for _, p := range packages {
						encodedB, err := s.SolverDatabase.CreatePackage(p)
						if err != nil {
							return err
						}
						B := bf.Var(encodedB)
						f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), bf.Not(B)))

						r := p.(*pkg.DefaultPackage) // We know since the implementation is DefaultPackage, that can be only DefaultPackage
						err = s.buildFormula(r, v, f)
						if err != nil {
							return err
						}
					}
					continue
				}
			}
		}

		encodedB, err := s.SolverDatabase.CreatePackage(required)
		if err != nil {
			return err
		}
		B := bf.Var(encodedB)
		f.Formulas = append(f.Formulas, bf.Or(bf.Not(A), bf.Not(B)))

		r := required.(*pkg.DefaultPackage) // We know since the implementation is DefaultPackage, that can be only DefaultPackage
		err = s.buildFormula(r, v, f)
		if err != nil {
			return err
		}
	}

	return nil
}

// BuildWorld builds the formula which olds the requirements from the package definitions
// which are available (global state)
func (s *SolverV2) BuildWorld(includeInstalled bool) (*bf.Formula, error) {
	start := time.Now()

	ff := &FormulasGate{
		Formulas: []bf.Formula{},
	}

	// NOTE: This block should be enabled in case of very old systems with outdated world sets
	if includeInstalled {
		err := s.BuildInstalled(ff)
		if err != nil {
			return nil, err
		}
	}

	Debug(fmt.Sprintf(
		"BuildWorld - after includeInstalled in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	m := make(map[string]interface{}, 0)
	for _, p := range s.World() {

		start2 := time.Now()
		err := s.buildFormula(p, &m, ff)
		if err != nil {
			return nil, err
		}

		Debug(fmt.Sprintf(
			"BuildWorld - after build formula pkg %s in %d (%d) µs (%d).",
			p.HumanReadableString(), time.Now().Sub(start).Nanoseconds()/1e3,
			time.Now().Sub(start2).Nanoseconds()/1e3,
			len(ff.Formulas)))
	}

	ans := bf.And(ff.Formulas...)

	Debug(fmt.Sprintf(
		"BuildWorld - after includeInstalled in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return &ans, nil
}

func (s *SolverV2) BuildInstalled(ff *FormulasGate) error {
	var packages pkg.Packages
	mInstalled := make(map[string]pkg.Packages, 0)
	m := make(map[string]interface{}, 0)

	for _, p := range s.Installed() {
		pp := pkg.Packages{}

		if val, ok := mInstalled[p.GetPackageName()]; ok {
			pp = *mergePackage(&val, p)
		} else {
			pp = append(pp, p)
		}

		mInstalled[p.GetPackageName()] = pp
	}

	for _, val := range mInstalled {
		packages = append(packages, val...)
	}

	fg := &FormulasGate{
		Formulas: []bf.Formula{},
	}

	for _, p := range packages {
		err := s.buildFormula(p, &m, fg)
		if err != nil {
			return err
		}
	}

	ff.Formulas = append(ff.Formulas, bf.And(fg.Formulas...))

	return nil
}

// BuildFormula builds the main solving formula that is evaluated by the sat solver.
func (s *SolverV2) BuildFormula() (*bf.Formula, error) {
	start := time.Now()
	var formulas []bf.Formula

	r, err := s.BuildWorld(false)
	if err != nil {
		return nil, err
	}

	Debug(fmt.Sprintf("BuildFormula - AFTER s.BuildWorld() - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	for _, wanted := range s.Wanted {

		encodedW, err := s.SolverDatabase.CreatePackage(wanted)
		if err != nil {
			return nil, err
		}

		W := bf.Var(encodedW)
		installedWorld := s.Installed()
		//TODO:Optimize
		if len(installedWorld) == 0 {
			formulas = append(formulas, W) //bf.And(bf.True, W))
			continue
		}

		for _, installed := range installedWorld {
			encodedI, err := s.SolverDatabase.CreatePackage(installed)
			if err != nil {
				return nil, err
			}
			I := bf.Var(encodedI)
			formulas = append(formulas, bf.And(W, I))
		}
	}

	Debug(fmt.Sprintf("BuildFormula - BEFORE bf.And() - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	formulas = append(formulas, *r)
	ans := bf.And(formulas...)

	return &ans, nil
}

// Solve builds the formula given the current state and returns package assertions
func (s *SolverV2) Solve() (PackagesAssertions, error) {
	var model map[string]bool
	var err error

	start := time.Now()

	f, err := s.BuildFormula()

	Debug(fmt.Sprintf("Solve - AFTER s.BuildFormula() - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	if err != nil {
		return nil, err
	}

	model = bf.Solve(*f)
	Debug(fmt.Sprintf("Solve - AFTER bf.Solve() - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	if model == nil {
		if s.Resolver != nil {
			return s.Resolver.Solve(*f, s)
		}
		return nil, errors.New("Unsolvable")
	}

	return DecodeModel(model, s.SolverDatabase)
}

func (s *SolverV2) relaxedInstall(c pkg.Packages) (PackagesAssertions, error) {
	start := time.Now()

	// TODO: copy takes time
	s.Wanted = c

	if s.noRulesWorld() {
		var ass PackagesAssertions
		for _, p := range s.Installed() {
			ass = append(ass, PackageAssert{Package: p.(*pkg.DefaultPackage), Value: true})

		}
		for _, p := range s.Wanted {
			ass = append(ass, PackageAssert{Package: p.(*pkg.DefaultPackage), Value: true})
		}
		return ass, nil
	}

	Debug(fmt.Sprintf("RelaxedInstall BEFORE s.Solve - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	assertions, err := s.Solve()
	if err != nil {
		return nil, err
	}

	Debug(fmt.Sprintf("RelaxedInstall AFTER s.Solve - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return assertions, nil
}

func (s *SolverV2) RelaxedInstall(c pkg.Packages) (PackagesAssertions, error) {
	coll, err := s.getList(s.DefinitionDatabase, c)
	if err != nil {
		return nil, errors.Wrap(err, "Packages not found in definition db")
	}

	return s.relaxedInstall(coll)
}

func (s *SolverV2) GetType() SolverType {
	return SingleCoreV2
}

func (s *SolverV2) Upgrade(checkconflicts, full bool) (pkg.Packages, PackagesAssertions, error) {
	// Hereinafter, the mission of the upgrade phase.
	// 1. Check if there are new versions of the installed packages.
	// 2. If there are new version, check if the new packages are installable
	// 3. Check if there are packages to remove.
	// 4. Build the formulas for SAT algorithm

	return s.upgrade(pkg.Packages{},
		pkg.Packages{},
		checkconflicts, full,
	)
}

func (s *SolverV2) computeUpgradeNew(ppsToUpgrade, ppsToNotUpgrade []pkg.Package, resp *UpgradeResponse) {
	toUninstall := pkg.Packages{}
	toInstall := pkg.Packages{}

	start := time.Now()
	for _, p := range s.InstalledDatabase.World() {
		packages, err := s.DefinitionDatabase.FindPackageVersions(p)

		if err == nil && len(packages) != 0 {
			best := packages.Best(nil)

			// This make sure that we don't try to upgrade something that was specified
			// specifically to not be marked for upgrade
			// At the same time, makes sure that if we mark a package to look for upgrades
			// it doesn't have to be in the blacklist (the packages to NOT upgrade)
			if !best.Matches(p) &&
				((len(ppsToUpgrade) == 0 && len(ppsToNotUpgrade) == 0) ||
					(inPackage(ppsToUpgrade, p) && !inPackage(ppsToNotUpgrade, p)) ||
					(len(ppsToUpgrade) == 0 && !inPackage(ppsToNotUpgrade, p))) {
				toUninstall = append(toUninstall, p)
				toInstall = append(toInstall, best)
			}
		}
	}

	Debug(fmt.Sprintf("computeUpgradeNew.for in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	resp.ToUninstall = &toUninstall
	resp.ToInstall = &toInstall
}

func (s *SolverV2) upgrade(psToUpgrade, psToNotUpgrade pkg.Packages,
	checkconflicts, full bool) (pkg.Packages, PackagesAssertions, error) {

	start := time.Now()

	installedcopy := pkg.NewInMemoryDatabase(false)
	err := s.InstalledDatabase.Clone(installedcopy)
	if err != nil {
		return nil, nil, err
	}

	Debug(fmt.Sprintf(
		"upgrade - installedDatabase.Clone in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	resp := NewUpgradeResponse()

	// Searching new versions of the installed packages
	s.computeUpgradeNew(psToUpgrade, psToNotUpgrade, resp)

	toUninstall := *resp.ToUninstall
	toInstall := *resp.ToInstall

	s2 := NewSolverV2(Options{Type: s.GetType()},
		installedcopy, s.DefinitionDatabase,
		pkg.NewInMemoryDatabase(false), s.Resolver)

	if !full {
		ass := PackagesAssertions{}
		for _, i := range toInstall {
			ass = append(ass, PackageAssert{Package: i.(*pkg.DefaultPackage), Value: true})
		}
	}

	Debug(fmt.Sprintf("Upgrade find uninstall %d, install %d", len(toUninstall), len(toInstall)))

	if len(toUninstall) == 0 && len(toInstall) == 0 {
		return pkg.Packages{}, PackagesAssertions{}, nil
	}

	// Then try to uninstall the versions in the system, and store that tree
	r, err := s.uninstall(checkconflicts, false, toUninstall...)
	if err != nil {
		return nil, nil, errors.Wrap(err,
			"Could not compute upgrade - couldn't uninstall candidates")
	}

	Debug(fmt.Sprintf("after s.uninstall - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	for _, z := range r {
		err = installedcopy.RemovePackage(z)
		if err != nil {
			return nil, nil, errors.Wrap(err,
				fmt.Sprintf(
					"Could not compute upgrade - couldn't remove copy of package %s targetted for removal",
					z.HumanReadableString(),
				))
		}
	}

	if len(toInstall) == 0 {
		ass := PackagesAssertions{}
		for _, i := range s.InstalledDatabase.World() {
			ass = append(ass, PackageAssert{Package: i.(*pkg.DefaultPackage), Value: true})
		}
		return toUninstall, ass, nil
	}

	assertions, err := s2.relaxedInstall(toInstall.Unique())

	Debug(fmt.Sprintf("upgrade - AFTER RelaxedInstall - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	// TODO: Check why check again the upgrade and if it's needed.
	//       Temporary i disable it.

	wantedSystem := assertions.ToDB()

	solvInstall := NewSolverV2(Options{Type: s.GetType()},
		wantedSystem, s.DefinitionDatabase,
		pkg.NewInMemoryDatabase(false), s.Resolver)

	Debug(fmt.Sprintf("upgrade - AFTER assertions.ToDB - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	resp = NewUpgradeResponse()
	if len(psToNotUpgrade) > 0 {
		// If we have packages in input,
		// compute what we are looking to upgrade.
		// those are assertions minus packsToUpgrade

		var selectedPackages []pkg.Package

		for _, p := range assertions {
			if p.Value && !inPackage(psToUpgrade, p.Package) {
				selectedPackages = append(selectedPackages, p.Package)
			}
		}
		solvInstall.computeUpgradeNew(selectedPackages, psToNotUpgrade, resp)
	} else {
		solvInstall.computeUpgradeNew(pkg.Packages{}, pkg.Packages{}, resp)
	}

	toInstall = *resp.ToInstall

	if len(toInstall) > 0 {

		_, ass, err := solvInstall.upgrade(
			psToUpgrade, psToNotUpgrade, checkconflicts, full,
		)

		Debug(fmt.Sprintf("fn.upgrade - solvInstall.upgrade in %d µs.",
			time.Now().Sub(start).Nanoseconds()/1e3))

		return toUninstall, ass, err
	}

	Debug(fmt.Sprintf("fn.upgrade - completed in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return toUninstall, assertions, err
}

func (s *SolverV2) uninstall(checkconflicts, full bool, expandedPacks ...pkg.Package) (pkg.Packages, error) {

	var res pkg.Packages
	start := time.Now()

	// Build a fake "Installed" - Candidate and its requires tree
	var InstalledMinusCandidate pkg.Packages

	// We are asked to not perform a full uninstall
	// (checking all the possible requires that could
	// be removed). Let's only check if we can remove the selected package
	if !full && checkconflicts {
		for _, candidate := range expandedPacks {
			if conflicts, err := s.Conflicts(candidate, s.Installed()); conflicts {
				return nil, errors.Wrap(err, "while searching for "+candidate.HumanReadableString()+" conflicts")
			}
		}
		return expandedPacks, nil
	}

	// TODO: Can be optimized
	for _, i := range s.Installed() {
		matched := false
		for _, candidate := range expandedPacks {
			if !i.Matches(candidate) {
				contains, err := candidate.RequiresContains(s.SolverDatabase, i)
				if err != nil {
					return nil, errors.Wrap(err, "Failed getting installed list")
				}
				if !contains {
					matched = true
				}

			}
		}
		if matched {
			InstalledMinusCandidate = append(InstalledMinusCandidate, i)
		}
	}

	s2 := NewSolverV2(Options{Type: s.GetType()},
		pkg.NewInMemoryDatabase(false),
		s.InstalledDatabase,
		pkg.NewInMemoryDatabase(false),
		s.Resolver,
	)

	// Get the requirements to install the candidate
	asserts, err := s2.relaxedInstall(expandedPacks)
	if err != nil {
		return nil, err
	}

	Debug(fmt.Sprintf("Uninstall AFTER s2.RelaxedInstall %d - - in %d µs.",
		len(asserts), time.Now().Sub(start).Nanoseconds()/1e3))

	for _, a := range asserts {
		if a.Value {
			if !checkconflicts {
				res = append(res, a.Package)
				continue
			}

			c, err := s.ConflictsWithInstalled(a.Package)
			if err != nil {
				return nil, err
			}

			// If doesn't conflict with installed we just consider it for removal and look for the next one
			if !c {
				res = append(res, a.Package)
				continue
			}

			// If does conflicts, give it another chance by checking conflicts if in case we
			// didn't installed our candidate and all the required packages in the system
			c, err = s.ConflictsWith(a.Package, InstalledMinusCandidate)
			if err != nil {
				return nil, err
			}
			if !c {
				res = append(res, a.Package)
			}

		}

	}

	Debug(fmt.Sprintf("Uninstall END - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return res, err
}

// Uninstall takes a candidate package and return a list of packages that would be removed
// in order to purge the candidate. Returns error if unsat.
func (s *SolverV2) Uninstall(checkconflicts, full bool,
	packs ...pkg.Package) (pkg.Packages, error) {

	// PRE: Uninstall receive the package without versions, i need
	// to expand the packages before go ahead.

	if len(packs) == 0 {
		return pkg.Packages{}, nil
	}

	start := time.Now()
	toRemove := pkg.Packages{}

	for _, c := range packs {
		candidate, err := s.InstalledDatabase.FindPackage(c)
		if err != nil {

			packages, err := c.Expand(s.InstalledDatabase)
			if err != nil || len(packages) == 0 {
				candidate = c
			} else {
				candidate = packages.Best(nil)
			}
			//Relax search, otherwise we cannot compute solutions for packages not in definitions
			//	return nil, errors.Wrap(err, "Package not found between installed")
		}

		toRemove = append(toRemove, candidate)
	}

	res, err := s.uninstall(checkconflicts, full, toRemove...)

	Debug(fmt.Sprintf("Uninstall PACKS - - in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return res, err
}

// Install returns the assertions necessary in order to install the packages in
// a system.
// It calculates the best result possible, trying to maximize new packages.
func (s *SolverV2) Install(c pkg.Packages) (PackagesAssertions, error) {
	assertions, err := s.RelaxedInstall(c)
	if err != nil {
		return nil, err
	}

	systemAfterInstall := pkg.NewInMemoryDatabase(false)

	toUpgrade := pkg.Packages{}
	toNotUpgrade := pkg.Packages{}
	for _, p := range c {
		if p.GetVersion() == ">=0" || p.GetVersion() == ">0" {
			toUpgrade = append(toUpgrade, p)
		} else {
			toNotUpgrade = append(toNotUpgrade, p)
		}
	}
	for _, p := range assertions {
		if p.Value {
			systemAfterInstall.CreatePackage(p.Package)
			if !inPackage(c, p.Package) && !inPackage(toUpgrade, p.Package) && !inPackage(toNotUpgrade, p.Package) {
				toUpgrade = append(toUpgrade, p.Package)
			}
		}
	}

	if len(toUpgrade) == 0 {
		return assertions, nil
	}

	resp := NewUpgradeResponse()
	s.computeUpgradeNew(toUpgrade, toNotUpgrade, resp)

	if len(*resp.ToUninstall) > 0 {
		// do partial upgrade based on input.
		// IF there is no version specified in the input, or >=0 is specified,
		// then compute upgrade for those
		_, newassertions, err := s.upgrade(toUpgrade, toNotUpgrade, false, false)
		if err != nil {
			// TODO: Emit warning.
			// We were not able to compute upgrades (maybe for some pinned packages, or a conflict)
			// so we return the relaxed result
			return assertions, nil
		}

		// Protect if we return no assertion at all
		if len(newassertions) == 0 && len(assertions) > 0 {
			return assertions, nil
		}
		return newassertions, nil
	}

	return assertions, nil
}

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
	"strings"
	//"fmt"
	//"time"

	//"github.com/pkg/errors"

	//	"github.com/crillab/gophersat/bf"
	"github.com/mudler/luet/pkg/config"
	pkg "github.com/mudler/luet/pkg/package"
)

// PackageSolver is an interface to a generic package solving algorithm
type PackageSolver interface {
	SetDefinitionDatabase(pkg.PackageDatabase)
	Install(p pkg.Packages) (PackagesAssertions, error)
	RelaxedInstall(p pkg.Packages) (PackagesAssertions, error)

	Uninstall(checkconflicts, full bool, candidate ...pkg.Package) (pkg.Packages, error)
	ConflictsWithInstalled(p pkg.Package) (bool, error)
	ConflictsWith(p pkg.Package, ls pkg.Packages) (bool, error)
	Conflicts(pack pkg.Package, lsp pkg.Packages) (bool, error)

	World() pkg.Packages
	Upgrade(checkconflicts, full bool) (pkg.Packages, PackagesAssertions, error)

	UpgradeUniverse(dropremoved bool) (pkg.Packages, PackagesAssertions, error)
	UninstallUniverse(toremove pkg.Packages) (pkg.Packages, error)

	SetResolver(PackageResolver)

	Solve() (PackagesAssertions, error)
	//	BestInstall(c pkg.Packages) (PackagesAssertions, error)

	GetType() SolverType
}

type SolverType int

const (
	SingleCoreSimple = 0
	SingleCoreV2     = 1
)

type Options struct {
	Type        SolverType `yaml:"type,omitempty"`
	Concurrency int        `yaml:"concurrency,omitempty"`
}

type UpgradeResponse struct {
	ToUninstall    *pkg.Packages
	ToInstall      *pkg.Packages
	InstalledCopy  *pkg.PackageDatabase
	PacksToUpgrade *[]pkg.Package
}

func (opts Options) ResolverIsSet() bool {
	switch config.LuetCfg.GetSolverOptions().Implementation {
	case QLearningResolverType:
		return true
	default:
		return false
	}
}

func (opts Options) Resolver() PackageResolver {
	switch config.LuetCfg.GetSolverOptions().Implementation {
	case QLearningResolverType:
		if config.LuetCfg.GetSolverOptions().LearnRate != 0.0 {
			return NewQLearningResolver(
				config.LuetCfg.GetSolverOptions().LearnRate,
				config.LuetCfg.GetSolverOptions().Discount,
				config.LuetCfg.GetSolverOptions().MaxAttempts,
				99999,
			)
		}
		return SimpleQLearningSolver()
	}

	return &Explainer{}
}

var AvailableResolvers = strings.Join([]string{
	QLearningResolverType,
	SolverV2ResolverType,
}, " ")

func NewUpgradeResponse() *UpgradeResponse {
	return &UpgradeResponse{}
}

func DecodeImplementation(i string) (ans Options) {
	switch i {
	case SolverV2ResolverType:
		ans.Type = SingleCoreV2
	default:
		ans.Type = SingleCoreSimple
	}

	ans.Concurrency = config.LuetCfg.GetGeneral().Concurrency
	return
}

// NewSolver accepts as argument two lists of packages, the first is the initial set,
// the second represent all the known packages.
func NewSolver(t Options, installed pkg.PackageDatabase, definitiondb pkg.PackageDatabase, solverdb pkg.PackageDatabase) PackageSolver {
	return NewResolver(t, installed, definitiondb, solverdb, &Explainer{})
}

// NewResolver accepts as argument two lists of packages, the first is the initial set,
// the second represent all the known packages.
// Using constructors as in the future we foresee warmups for hot-restore solver cache
func NewResolver(t Options, installed pkg.PackageDatabase, definitiondb pkg.PackageDatabase, solverdb pkg.PackageDatabase, re PackageResolver) PackageSolver {
	var s PackageSolver
	switch t.Type {
	case SingleCoreV2:
		s = NewSolverV2(t, installed, definitiondb, solverdb, re)

	default:
		s = &Solver{
			InstalledDatabase:  installed,
			DefinitionDatabase: definitiondb,
			SolverDatabase:     solverdb,
			Resolver:           re,
		}
	}

	return s
}

func inPackage(list []pkg.Package, p pkg.Package) bool {
	for _, l := range list {
		if l.AtomMatches(p) {
			return true
		}
	}
	return false
}

func mergePackage(list *pkg.Packages, p pkg.Package) *pkg.Packages {

	ans := pkg.Packages{}
	present := false

	for _, e := range *list {
		if e.Matches(p) {
			present = true
		}
		ans = append(ans, e)
	}

	if !present {
		ans = append(ans, p)
	}

	return &ans
}

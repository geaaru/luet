/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/macaroni-os/anise/pkg/config"
	"github.com/macaroni-os/anise/pkg/helpers"
	. "github.com/macaroni-os/anise/pkg/logger"
	pkg "github.com/macaroni-os/anise/pkg/package"
	"github.com/macaroni-os/anise/pkg/v2/compiler/types/artifact"
	"github.com/macaroni-os/anise/pkg/v2/compiler/types/specs"
	"github.com/macaroni-os/anise/pkg/v2/render"
	"github.com/macaroni-os/anise/pkg/v2/tree"

	. "github.com/logrusorgru/aurora"
)

type BuildSolver struct {
	Config *config.LuetConfig `yaml:",inline" json:",inline"`

	ForestGuard  *tree.ForestGuard    `yaml:"-" json:"-"`
	RenderEngine *render.RenderEngine `yaml:"-" json:"-"`
	Opts         *BuildSolverOpts     `yaml:"-" json:"-"`

	CacheMap *artifact.ArtifactsMap `yaml:"-" json:"-"`

	mutex *sync.Mutex `yaml:"-" json:"-"`
}

func NewBuildSolver(cfg *config.LuetConfig,
	opts *BuildSolverOpts) *BuildSolver {
	return &BuildSolver{
		Config:   cfg,
		Opts:     opts,
		CacheMap: artifact.NewArtifactsMap(),
		mutex:    &sync.Mutex{},
	}
}

func (s *BuildSolver) ClearCache() {
	s.CacheMap = artifact.NewArtifactsMap()
}

func (s *BuildSolver) GetForestGuard() *tree.ForestGuard { return s.ForestGuard }
func (s *BuildSolver) SetForestGuard(g *tree.ForestGuard) {
	s.ForestGuard = g
}

func (s *BuildSolver) SetRenderEngine(re *render.RenderEngine) { s.RenderEngine = re }
func (s *BuildSolver) GetRenderEngine() *render.RenderEngine   { return s.RenderEngine }

func (s *BuildSolver) Resolve(pkgs *[]*pkg.DefaultPackage) (*artifact.ArtifactsPack, error) {
	ans := artifact.NewArtifactsPack()

	// As for

	return ans, nil
}

func (s *BuildSolver) ResolvePackage(p *pkg.DefaultPackage) (*artifact.ArtifactsPack, error) {
	ans := artifact.NewArtifactsPack()
	apMap := artifact.NewArtifactsMap()
	vMap := make(map[string]bool, 0)

	InfoC(Bold(fmt.Sprintf(":eyes: Resolving selector...               %s",
		p.HumanReadableString())))

	// Retrieve all versions matched by selector
	indexes, err := s.ForestGuard.SearchPackage(p)
	if err != nil {
		return ans, err
	}

	if len(indexes) == 0 {
		return ans, fmt.Errorf("No candidates for selector %s found.",
			p.HumanReadableString())
	}

	for _, ti := range indexes {
		// POST: the indexes contains one or more TreeIdx and
		//       in every TreeIdx we have one or more versions

		versions, present := ti.GetPackageVersions(p.PackageName())
		if !present {
			return ans, fmt.Errorf(
				"unexpected error on retrieve versions for package %s and tree with basedir %s",
				p.PackageName(), ti.BaseDir)
		}

		for _, tv := range versions {

			if _, processed := vMap[tv.Version]; processed {
				// Avoid to elaborate the same version
				// multiple times. Using always the first.
				continue
			}
			vMap[tv.Version] = true

			Debug(fmt.Sprintf(":construction: Creating package task for %s-%s",
				p.PackageName(), tv.Version))
			ptask := NewPackageTask(ti, tv, p)
			pack, err := s.ResolvePackageTask(ptask)
			if err != nil {
				return ans, err
			}

			// Add only packages not yet injected
			for _, part := range pack.Artifacts {
				_, missed := apMap.MatchVersion(part)
				if missed != nil {
					apMap.Add(part)
				} // else package is alreade present.
			}
		}
	}

	ans.Artifacts = *apMap.ToList()

	return ans, nil
}

func (s *BuildSolver) loadCompilationSpec(
	t *tree.TreeIdx, vtree *tree.TreeIdxPkg, p *pkg.DefaultPackage) (*specs.CompilationSpecLoad, string, error) {

	var cs *specs.CompilationSpecLoad
	var err error

	// Using render engine to read build.yaml
	pkgPath := filepath.Join(t.TreePath, t.BaseDir,
		filepath.Dir(vtree.Path))

	defFile := filepath.Join(pkgPath, filepath.Base(vtree.Path))
	buildFile := filepath.Join(pkgPath, "build.yaml")

	DebugC(fmt.Sprintf(":brain:For %s-%s using buildfile:\t%s",
		p.PackageName(), vtree.Version, buildFile))

	DebugC(fmt.Sprintf(
		":brain:For %s-%s using package specs:\t%s",
		p.PackageName(), vtree.Version, defFile))

	if filepath.Base(defFile) == "collection.yaml" {

		atom := pkg.NewPackageWithCatThin(
			p.Category, p.Name,
			vtree.Version)

		cs, err = tree.ReadBuildFileFromCollection(buildFile, defFile,
			s.RenderEngine, atom, map[string]interface{}{})
	} else {
		cs, err = tree.ReadBuildFile(buildFile, defFile,
			s.RenderEngine, map[string]interface{}{})
	}
	if err != nil {
		return nil, "", fmt.Errorf(
			"error on rendering package %s-%s: %s",
			p.PackageName(), vtree.Version, err.Error())
	}

	return cs, pkgPath, nil
}

func (s *BuildSolver) ResolvePackageTask(ptask *PackageTask) (*artifact.ArtifactsPack, error) {
	// The stack array is used to catch dependencies cycles.
	stack := []string{}

	// Resolve recors
	return s.resolvePackage(ptask, stack)
}

// func (s *BuildSolver) recursiveResolveDep(ptask *PackageTask, stack []string)

func (s *BuildSolver) resolvePackage(ptask *PackageTask, stack []string) (*artifact.ArtifactsPack, error) {
	ans := artifact.NewArtifactsPack()

	if helpers.ContainsElem(&stack, ptask.PackageSelector.PackageName()) {
		// POST: this package is already been elaboratored. Stop dep cycle.
		return ans, nil
	}

	InfoC(fmt.Sprintf(":dart: :right_arrow: Elaborating package...           %s",
		Bold(fmt.Sprintf("%s-%s",
			ptask.PackageSelector.PackageName(), ptask.Version.Version))))

	stack = append(stack, ptask.PackageSelector.PackageName())

	cs, pkgPath, err := s.loadCompilationSpec(ptask.Tree, ptask.Version, ptask.PackageSelector)
	if err != nil {
		return ans, err
	}

	// Stage1. Before elaborate all dependencies I try to retrieve all dependencies
	//         selectors to validate AND conditions.

	// Check if the package has conflicts
	if cs.DefaultPackage != nil && len(cs.DefaultPackage.GetConflicts()) > 0 {
		for _, conflict := range cs.DefaultPackage.GetConflicts() {
			ptask.AddConflict(conflict)
		}
	}

	// Check if the package has dependencies to recursively resolve.
	if cs.DefaultPackage != nil && len(cs.DefaultPackage.GetRequires()) > 0 {

		for _, dep := range cs.DefaultPackage.GetRequires() {
			// Retrieve all available packages of the selected dependency
			reqIdx, err := s.ForestGuard.SearchPackage(dep)
			if err != nil {
				return ans, err
			}

			// Fragments and sort all availables version. (it drops duplicates too).
			// Sort in reverse order (newest before old).
			reqIdx = *tree.FragmentTrees(&reqIdx, dep.PackageName(), true)

			ptask.AddDependency(NewDependencySelectorWithIdx(dep, &reqIdx))

			// Iterate for every version available of the dependency. I will
			// add informations
			//
		}
	}

	// Create Package artifact with path sets to the home directory
	// for the build.
	ptask.Artifact = artifact.NewPackageArtifact(pkgPath)
	ptask.Artifact.CompileSpec = cs.ToSpec()

	// Setup selected dependencies on final Artifact object.
	if len(ptask.candidatesDepsMap.Artifacts) > 0 {

	} // else no dependencies for building.

	// Add at the end of the list the package to build
	ans.Add(ptask.Artifact)

	return ans, nil
}

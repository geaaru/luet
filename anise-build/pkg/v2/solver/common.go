/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"github.com/macaroni-os/anise/pkg/config"
	pkg "github.com/macaroni-os/anise/pkg/package"
	"github.com/macaroni-os/anise/pkg/v2/compiler/types/artifact"
	"github.com/macaroni-os/anise/pkg/v2/render"
	"github.com/macaroni-os/anise/pkg/v2/tree"
)

type BuildSolverOpts struct {
	IgnoreConflicts bool
	Force           bool
	NoDeps          bool
}

type BuildSolverType int

type BuilderSolver interface {
	SetForestGuard(g *tree.ForestGuard)
	GetForestGuard() *tree.ForestGuard
	SetRenderEngine(re *render.RenderEngine)
	GetRenderEngine() *render.RenderEngine
	Resolve(pkg *[]*pkg.DefaultPackage) (*artifact.ArtifactsPack, error)
	ResolvePackage(pkg *pkg.DefaultPackage) (*artifact.ArtifactsPack, error)
	ResolvePackageTask(ptask *PackageTask) (*artifact.ArtifactsPack, error)
}

func NewBuildSolverOpts() *BuildSolverOpts {
	return &BuildSolverOpts{
		IgnoreConflicts: false,
		Force:           false,
		NoDeps:          false,
	}
}

func NewBuildSolverImplementation(
	stype string,
	cfg *config.LuetConfig,
	opts *BuildSolverOpts) *BuilderSolver {
	var s BuilderSolver

	switch stype {
	default:
		s = NewBuildSolver(cfg, opts)
	}

	return &s
}

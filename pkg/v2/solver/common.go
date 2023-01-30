/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"github.com/geaaru/luet/pkg/config"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
)

type SolverType int

const (
	SingleCoreSimple = 0
	SingleCoreV2     = 1
	SingleCoreV3     = 2
)

const (
	UpdatePackage = "U"
	AddPackage    = "N"
	RemovePackage = "D"
)

type SolverOpts struct {
	IgnoreConflicts bool
	NoDeps          bool
}

type Operation struct {
	Action   string                    `yaml:"action" json:"action"`
	Artifact *artifact.PackageArtifact `yaml:"artefact" json:"artefact"`
}

func NewSolverOpts() *SolverOpts {
	return &SolverOpts{
		IgnoreConflicts: false,
		NoDeps:          false,
	}
}

// PackageSolver is an interface to a generic package solving algorithm
type PackageSolver interface {
	Install(p *[]*pkg.DefaultPackage) (*artifact.ArtifactsPack, *artifact.ArtifactsPack, error)
	Upgrade() (*artifact.ArtifactsPack, *artifact.ArtifactsPack, *artifact.ArtifactsPack, error)
	GetType() SolverType
	SetDatabase(pkg.PackageDatabase)
	OrderOperations(p2i, p2u, p2r *artifact.ArtifactsPack) (*[]*Operation, error)
}

func NewOperation(action string, art *artifact.PackageArtifact) *Operation {
	return &Operation{
		Action:   action,
		Artifact: art,
	}
}

func NewSolverImplementation(stype string, cfg *config.LuetConfig, opts *SolverOpts) *PackageSolver {
	var s PackageSolver

	switch stype {
	case "solverv2":
		// TODO: For now remap all implementation to the new solver implementation.
		s = NewSolver(cfg, opts)
	default:
		s = NewSolver(cfg, opts)
	}

	return &s
}

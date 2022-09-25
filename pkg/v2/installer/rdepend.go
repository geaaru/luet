/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"sync"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/pkg/errors"
)

type ResolveRdependsTask struct {
	resolvedMap map[string]bool

	Package        *pkg.DefaultPackage
	System         *pkg.Packages
	SystemRdepsMap map[string][]*pkg.DefaultPackage

	Matches []*pkg.DefaultPackage
	Mutex   sync.Mutex
}

func NewResolveRdependsTask() *ResolveRdependsTask {
	return &ResolveRdependsTask{
		resolvedMap:    make(map[string]bool, 0),
		Package:        nil,
		System:         nil,
		SystemRdepsMap: make(map[string][]*pkg.DefaultPackage, 0),
		Matches:        []*pkg.DefaultPackage{},
	}
}

func (r *ResolveRdependsTask) SetResolved(p *pkg.DefaultPackage) {
	r.resolvedMap[p.PackageName()] = true
}

func (r *ResolveRdependsTask) IsResolved(p *pkg.DefaultPackage) bool {
	_, ok := r.resolvedMap[p.PackageName()]
	return ok
}

func (r *ResolveRdependsTask) AddRdep2Map(p *pkg.DefaultPackage, d *pkg.DefaultPackage) {
	if val, ok := r.SystemRdepsMap[d.PackageName()]; ok {
		r.SystemRdepsMap[d.PackageName()] = append(
			val, p,
		)

	} else {
		r.SystemRdepsMap[d.PackageName()] = []*pkg.DefaultPackage{p}
	}
}

func (r *ResolveRdependsTask) AddMatch(p *pkg.DefaultPackage) {
	r.Matches = append(r.Matches, p)
}

func (m *ArtifactsManager) ResolveRuntime(task *ResolveRdependsTask) error {

	task.SystemRdepsMap = make(map[string][]*pkg.DefaultPackage, 0)

	// TEMPORARY solution that require a lot of memory.
	// Create rdeps map
	for _, p := range *task.System {

		provides := p.GetProvides()
		requires := p.GetRequires()

		Debug(fmt.Sprintf(
			"[%s] provides %d\n[%s] requires %d",
			p.HumanReadableString(), len(provides),
			p.HumanReadableString(), len(requires)))

		if len(provides) > 0 {
			for idx, _ := range provides {
				Debug(fmt.Sprintf("[%s] Add provide %s",
					p.HumanReadableString(), provides[idx].HumanReadableString()))
				task.AddRdep2Map(p.(*pkg.DefaultPackage), provides[idx])
			}
		}

		if len(requires) > 0 {
			for idx, _ := range requires {
				Debug(fmt.Sprintf("[%s] Add require %s",
					p.HumanReadableString(), requires[idx].HumanReadableString()))
				task.AddRdep2Map(p.(*pkg.DefaultPackage), requires[idx])
			}
		}
	}

	err := m.recursiveResolveRdep(task.Package, task)
	if err != nil {
		return err
	}

	return nil
}

func (m *ArtifactsManager) recursiveResolveRdep(
	p *pkg.DefaultPackage,
	task *ResolveRdependsTask) error {

	// Retrieve all packages using the package in input
	if task.IsResolved(p) {
		Debug(fmt.Sprintf("Found package %s already resolved.", p.PackageName()))
		return nil
	}
	// Set dependency as resolved
	task.SetResolved(p)
	task.AddMatch(p)

	if val, ok := task.SystemRdepsMap[p.PackageName()]; ok {

		for _, d := range val {
			if !task.IsResolved(d) {
				err := m.recursiveResolveRdep(d, task)
				if err != nil {
					return errors.New(
						fmt.Sprintf("Error on resolve dependency %s: %s",
							d.PackageName(), err.Error()))
				}
			}
			// else the dependency is already been resolved
			// and injected.
		}
	}
	// else Nothing to do

	return nil
}

/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"sort"

	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
)

func SortArtifactList4RequiresAndRepos(artsref *[]*artifact.PackageArtifact,
	rmap *map[string]*WagonRepository) {

	arts := *artsref
	mapRepos := *rmap

	// Sort packages ordered for number of requires
	// and for repository priority
	sort.Slice(arts[:], func(i, j int) bool {

		arti := arts[i]
		artj := arts[j]

		pi := arti.GetPackage()
		pj := artj.GetPackage()

		ireq := pi.HasRequires()
		jreq := pj.HasRequires()

		// NOTE: If i don't find the repository in the
		//       map i consider a priority with value 100
		irprio := 100
		jrprio := 100

		if r, ok := mapRepos[pi.Repository]; ok {
			irprio = r.Identity.Priority
		}
		if r, ok := mapRepos[pj.Repository]; ok {
			jrprio = r.Identity.Priority
		}

		if ireq && jreq {
			if len(pi.PackageRequires) == len(pj.PackageRequires) {
				return irprio < jrprio
			}
			return len(pi.PackageRequires) < len(pj.PackageRequires)
		} else if !ireq && !jreq {
			return irprio < jrprio
		} else if !ireq {
			return true
		}
		return false
	})
}

// Sort packages ordered for repository priority,
// for number of requires and for PackageName()
func SortArtifactList4ReposAndRequires(artsref *[]*artifact.PackageArtifact,
	rmap *map[string]*WagonRepository) {

	arts := *artsref
	mapRepos := *rmap

	// Sort packages ordered for number of requires
	// and for repository priority
	sort.Slice(arts[:], func(i, j int) bool {

		arti := arts[i]
		artj := arts[j]

		pi := arti.GetPackage()
		pj := artj.GetPackage()

		ireq := pi.HasRequires()
		jreq := pj.HasRequires()

		// NOTE: If i don't find the repository in the
		//       map i consider a priority with value 100
		irprio := 100
		jrprio := 100

		if r, ok := mapRepos[pi.Repository]; ok {
			irprio = r.Identity.Priority
		}
		if r, ok := mapRepos[pj.Repository]; ok {
			jrprio = r.Identity.Priority
		}

		if irprio == jrprio {
			if ireq && jreq {
				if len(pi.PackageRequires) == len(pj.PackageRequires) {
					return pi.PackageName() < pj.PackageName()
				}
				return len(pi.PackageRequires) < len(pj.PackageRequires)
			} else if !ireq && !jreq {
				return pi.PackageName() < pj.PackageName()
			} else if !ireq {
				return true
			}
		} else {
			return irprio < jrprio
		}

		return false
	})
}

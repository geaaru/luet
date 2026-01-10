/*
Copyright © 2023-2024 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"fmt"
	"strings"

	cfg "github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	pkg "github.com/geaaru/luet/pkg/package"

	_gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

// The ForestGuard object is used to
// analize and iterate over TreeIdx objects.
type ForestGuard struct {
	Config *cfg.LuetConfig
	Trees  []*TreeIdx
}

func NewForestGuard(config *cfg.LuetConfig) *ForestGuard {
	return &ForestGuard{
		Config: config,
		Trees:  []*TreeIdx{},
	}
}

func (fg *ForestGuard) LoadTrees(tpaths []string) error {
	for _, t := range tpaths {
		tIdx := NewTreeIdx(t, true).DetectMode()
		err := tIdx.Read(t)
		if err != nil {
			return err
		}

		fg.Trees = append(fg.Trees, tIdx)
	}

	return nil
}

func (fg *ForestGuard) resolveCategory(name string) (string, error) {
	matches := []string{}
	ans := ""

	for _, ti := range fg.Trees {
		for k, _ := range ti.Map {
			words := strings.Split(k, "/")
			if words[1] == name {
				ans = words[0]
				if len(matches) == 0 {
					matches = append(matches, k)
				} else if len(matches) > 1 && matches[0] != k {
					return "", fmt.Errorf("multiple matches for %s:\n%s",
						name, strings.Join(matches, ","))
				}
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no matching for package with name %s",
			name)
	}

	return ans, nil
}

func (fg *ForestGuard) SearchPackage(p *pkg.DefaultPackage) ([]*TreeIdx, error) {
	ans := []*TreeIdx{}

	gps, _ := p.ToGentooPackage()

	for _, ti := range fg.Trees {

		versions, present := ti.GetPackageVersions(p.PackageName())
		if !present {
			continue
		}

		tProcessed := NewTreeIdx(ti.TreePath, ti.Compress)
		tProcessed.BaseDir = ti.BaseDir

		for _, ver := range versions {
			pkg2check := &pkg.DefaultPackage{
				Name:     p.Name,
				Category: p.Category,
				Version:  ver.Version,
			}

			gp, _ := pkg2check.ToGentooPackage()
			admitted, err := gps.Admit(gp)
			if err != nil {
				return ans, err
			}

			if admitted {
				tProcessed.AddPackage(pkg2check.PackageName(), ver)
			}
		}

		if tProcessed.HasPackages() {
			ans = append(ans, tProcessed)
		}
	}

	return ans, nil
}

func (fg *ForestGuard) ResolveSelector(p string) (*pkg.DefaultPackage, error) {
	var err error
	ver := ">=0"
	cat := ""
	name := ""
	cond := _gentoo.PackageCond(_gentoo.PkgCondInvalid)

	if strings.Contains(p, "@") || !strings.Contains(p, "/") {

		pinfo := strings.Split(p, "@")
		if len(pinfo) > 1 {
			ver = pinfo[1]
		}
		cat, name, cond = helpers.PackageResolveSplit(p)

		if cond != _gentoo.PkgCondInvalid && ver != "" {
			ver = cond.String() + ver
		}

		if cat == "" {
			// POST: searching between all keys of the indexes a match of
			//       the name.
			cat, err = fg.resolveCategory(name)
			if err != nil {
				return nil, err
			}
		}

	} else {
		// POST: package string contains / but not @

		gp, err := _gentoo.ParsePackageStr(p)
		if err != nil {
			return nil, err
		}

		if gp.Version == "" {
			gp.Version = "0"
			gp.Condition = _gentoo.PkgCondGreaterEqual
		}

		ver = helpers.GentooVersion(gp)
		cat = gp.Category
		name = gp.Name
	}

	ans := &pkg.DefaultPackage{
		Name:     name,
		Category: cat,
		Version:  ver,
		Uri:      make([]string, 0),
	}

	return ans, nil
}

func (fg *ForestGuard) Search(p string) ([]*TreeIdx, error) {
	ans := []*TreeIdx{}

	pkg2search, err := fg.ResolveSelector(p)
	if err != nil {
		return ans, err
	}

	return fg.SearchPackage(pkg2search)
}

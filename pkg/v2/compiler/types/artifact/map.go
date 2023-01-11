/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package artifact

import (
	"errors"
	"fmt"
	"sort"

	gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

type ArtifactsMap struct {
	Artifacts map[string][]*PackageArtifact `json:"artifacts" yaml:"artifacts"`
}

type ArtifactsPack struct {
	Artifacts []*PackageArtifact `json:"artifacts" yaml:"artifacts"`
}

func NewArtifactsMap() *ArtifactsMap {
	return &ArtifactsMap{
		Artifacts: make(map[string][]*PackageArtifact, 0),
	}
}

func NewArtifactsPack() *ArtifactsPack {
	return &ArtifactsPack{
		Artifacts: []*PackageArtifact{},
	}
}

func (ap *ArtifactsPack) ToMap() *ArtifactsMap {
	ans := &ArtifactsMap{
		Artifacts: make(map[string][]*PackageArtifact, 1),
	}

	for idx, _ := range ap.Artifacts {
		ans.Add(ap.Artifacts[idx])
	}

	return ans
}

func (am *ArtifactsMap) MatchVersion(p *PackageArtifact) (*PackageArtifact, error) {
	var ans *PackageArtifact = nil
	var key string

	if p.Runtime != nil {
		key = p.Runtime.PackageName()
	} else {
		key = p.CompileSpec.Package.PackageName()
	}

	if val, ok := am.Artifacts[key]; ok {
		for idx, _ := range val {
			pp := val[idx]

			if pp.GetVersion() == p.GetVersion() {
				ans = pp
				break
			}
		}
	}

	if ans == nil {
		return ans, errors.New(fmt.Sprintf("Package %s-%s not found", key, p.GetVersion()))
	}

	return ans, nil
}

func (am *ArtifactsMap) Add(p *PackageArtifact) {
	var key string
	if p.Runtime != nil {
		key = p.Runtime.PackageName()
	} else {
		key = p.CompileSpec.Package.PackageName()
	}

	if val, ok := am.Artifacts[key]; ok {
		am.Artifacts[key] = append(val, p)
	} else {
		am.Artifacts[key] = []*PackageArtifact{p}
	}
}

func (am *ArtifactsMap) GetSortedArtifactsByKey(k string) ([]*PackageArtifact, error) {
	ans := []*PackageArtifact{}

	val, ok := am.Artifacts[k]
	if !ok {
		panic(errors.New(fmt.Sprintf("Package %s not found", k)))
		return nil, errors.New(fmt.Sprintf("Package %s not found on map", k))
	}

	if len(val) == 1 {
		ans = val
	} else {

		// NOTE: In the near future the DefaultPackage will be based
		//       on GentooPackage implementation and I will reduce complexity
		//       here. For now, I convert the list on GentooPackageSorter

		// TODO: At the moment I don't respect the repository priority for packages
		//       with the same version but I have not repositories data here.
		glist := []gentoo.GentooPackage{}

		for _, p := range val {
			gpkg, err := gentoo.ParsePackageStr(p.GetPackage().HumanReadableString())
			if err != nil {
				return nil, err
			}
			glist = append(glist, *gpkg)
		}

		sort.Sort(sort.Reverse(gentoo.GentooPackageSorter(glist)))

		for idx, _ := range glist {
			pstr := glist[idx].String()
			if glist[idx].VersionBuild != "" {
				pstr = fmt.Sprintf("%s+%s", pstr, glist[idx].VersionBuild)
			}
			for _, p := range val {
				if p.GetPackage().HumanReadableString() == pstr {
					ans = append(ans, p)
					break
				}
			}
		}

	}

	return ans, nil
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package artifact

import (
	"errors"
	"fmt"
)

type ArtifactsMap struct {
	Artifacts map[string][]*PackageArtifact `json:"artifacts" yaml:"artifacts"`
}

type ArtifactsPack struct {
	Artifacts []*PackageArtifact `json:"artifacts" yaml:"artifacts"`
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

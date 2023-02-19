/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package pkg

type PkgsMapList struct {
	Packages map[string][]*DefaultPackage `json:"packages_map" yaml:"packages_map"`
}

func NewPkgsMapList() *PkgsMapList {
	return &PkgsMapList{
		Packages: make(map[string][]*DefaultPackage, 0),
	}
}

func (pm *PkgsMapList) Add(k string, p *DefaultPackage) {
	if val, ok := pm.Packages[k]; ok {
		pm.Packages[k] = append(val, p)
	} else {
		pm.Packages[k] = []*DefaultPackage{p}
	}
}

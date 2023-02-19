/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package pkg

type PkgsList struct {
	Packages []*DefaultPackage `json:"packages" yaml:"packages"`
}

func NewPkgsList(list *[]*DefaultPackage) *PkgsList {
	return &PkgsList{
		Packages: *list,
	}
}

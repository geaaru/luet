/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package pkg

type PackagesMap map[string]*DefaultPackage

func (p *Packages) ToMap() *map[string]*DefaultPackage {
	ans := make(map[string]*DefaultPackage, 0)

	for _, p := range *p {
		ans[p.PackageName()] = p.(*DefaultPackage)
	}

	return &ans
}

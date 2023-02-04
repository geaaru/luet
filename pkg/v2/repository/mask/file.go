/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package mask

import (
	"errors"

	gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
	"gopkg.in/yaml.v2"
)

func NewPackageMaskFile(file string) *PackageMaskFile {
	return &PackageMaskFile{
		File:    file,
		Enabled: true,
		Rules:   []string{},
		pkgsMap: make(map[string][]*gentoo.GentooPackage, 0),
	}
}

func NewPackageMaskFileFromData(file string, data []byte) (*PackageMaskFile, error) {
	ans := NewPackageMaskFile(file)
	err := yaml.Unmarshal(data, ans)
	if err != nil {
		return nil, err
	}
	return ans, nil
}

func (f *PackageMaskFile) BuildMap() error {
	for _, rule := range f.Rules {
		p, err := gentoo.ParsePackageStr(rule)
		if err != nil {
			return err
		}

		pkgstr := f.getPkgStr(p)
		if val, ok := f.pkgsMap[pkgstr]; ok {
			f.pkgsMap[pkgstr] = append(val, p)
		} else {
			f.pkgsMap[pkgstr] = []*gentoo.GentooPackage{p}
		}
	}

	return nil
}

func (f *PackageMaskFile) getPkgStr(p *gentoo.GentooPackage) string {
	ans := p.Category
	if p.Slot != "" && p.Slot != "0" {
		ans += "-" + p.Slot
	}
	ans += "/" + p.Name

	return ans
}

func (f *PackageMaskFile) Mask(repo string, p *gentoo.GentooPackage) (bool, error) {
	if p == nil {
		return false, errors.New("Invalid package")
	}

	// Check if the package p is available in the map
	pkgstr := f.getPkgStr(p)
	if pkgs, ok := f.pkgsMap[pkgstr]; ok {
		for _, pm := range pkgs {
			if pm.Repository != "" && pm.Repository != repo {
				// Ignoring mask related to a different repository.
				continue
			}

			admit, err := pm.Admit(p)
			if err != nil {
				return false, err
			}

			if admit {
				// POST: The package in input match with the existing
				//       rule selector.
				return true, nil
			}
		}
	}
	// POST: The package is not masked.

	return false, nil
}

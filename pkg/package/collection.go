/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package pkg

import (
	"fmt"
)

func (c *Collection) GetPackage(pkgName, version string) (*DefaultPackage, error) {
	for idx, p := range c.Packages {
		if p.PackageName() == pkgName && p.GetVersion() == version {
			return &c.Packages[idx], nil
		}
	}

	return nil, fmt.Errorf("Package %s-%s not in collection", pkgName, version)
}

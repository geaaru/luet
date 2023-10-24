/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"fmt"
	"strings"

	_gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

func PackageResolveSplit(p string) (string, string, _gentoo.PackageCond) {
	cat := ""
	name := ""
	cond := _gentoo.PackageCond(_gentoo.PkgCondInvalid)

	if strings.Contains(p, "/") {
		gp, _ := _gentoo.ParsePackageStr(p)
		cond = gp.Condition
		cat = gp.Category
		name = gp.Name
	} else {
		name = p
	}
	return cat, name, cond
}

func GentooVersion(gp *_gentoo.GentooPackage) string {
	condition := gp.Condition.String()
	if condition == "=" {
		condition = ""
	}

	pkgVersion := fmt.Sprintf("%s%s%s",
		condition,
		gp.Version,
		gp.VersionSuffix,
	)
	if gp.VersionBuild != "" {
		pkgVersion = fmt.Sprintf("%s%s%s+%s",
			condition,
			gp.Version,
			gp.VersionSuffix,
			gp.VersionBuild,
		)
	}
	return pkgVersion
}

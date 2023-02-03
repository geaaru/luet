/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd_helpers

import (
	"fmt"
	"strings"

	. "github.com/geaaru/luet/pkg/logger"

	pkg "github.com/geaaru/luet/pkg/package"
	_gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

func packageData(p string) (string, string, _gentoo.PackageCond) {
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

func gentooVersion(gp *_gentoo.GentooPackage) string {

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

func ParsePackageStr(p string) (*pkg.DefaultPackage, error) {
	ver := ">=0"
	cat := ""
	name := ""
	cond := _gentoo.PackageCond(_gentoo.PkgCondInvalid)

	if strings.Contains(p, "@") || !strings.Contains(p, "/") {
		packageinfo := strings.Split(p, "@")
		if len(packageinfo) > 1 {
			ver = packageinfo[1]
		}
		cat, name, cond = packageData(packageinfo[0])
		if cond != _gentoo.PkgCondInvalid && ver != "" {
			ver = cond.String() + ver
		}
	} else {
		gp, err := _gentoo.ParsePackageStr(p)
		if err != nil {
			return nil, err
		}

		if gp.Version == "" {
			gp.Version = "0"
			gp.Condition = _gentoo.PkgCondGreaterEqual
		}

		ver = gentooVersion(gp)
		cat = gp.Category
		name = gp.Name
	}

	return &pkg.DefaultPackage{
		Name:     name,
		Category: cat,
		Version:  ver,
		Uri:      make([]string, 0),
	}, nil
}

func CheckErr(err error) {
	if err != nil {
		Fatal(err)
	}
}

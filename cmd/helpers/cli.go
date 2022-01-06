// Copyright © 2020 Ettore Di Giacinto <mudler@gentoo.org>
//                  Daniele Rondina <geaaru@sabayonlinux.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package cmd_helpers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	. "github.com/mudler/luet/pkg/logger"

	_gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
	pkg "github.com/mudler/luet/pkg/package"
)

func CreateRegexArray(rgx []string) ([]*regexp.Regexp, error) {
	ans := make([]*regexp.Regexp, len(rgx))
	if len(rgx) > 0 {
		for idx, reg := range rgx {
			re := regexp.MustCompile(reg)
			if re == nil {
				return nil, errors.New("Invalid regex " + reg + "!")
			}
			ans[idx] = re
		}
	}

	return ans, nil
}

func packageData(p string) (string, string) {
	cat := ""
	name := ""
	if strings.Contains(p, "/") {
		packagedata := strings.Split(p, "/")
		cat = packagedata[0]
		name = packagedata[1]
	} else {
		name = p
	}
	return cat, name
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

	if strings.Contains(p, "@") || !strings.Contains(p, "/") {
		packageinfo := strings.Split(p, "@")
		if len(packageinfo) > 1 {
			ver = packageinfo[1]
		}
		cat, name = packageData(packageinfo[0])
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

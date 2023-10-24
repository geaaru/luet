/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package cmd_helpers

import (
	"errors"
	"fmt"
	"strings"

	cfg "github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	wagon "github.com/geaaru/luet/pkg/v2/repository"

	_gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

func resolveCategory(config *cfg.LuetConfig, name string) (string, error) {
	ans := ""

	searchOpts := &wagon.StonesSearchOpts{
		Names:        []string{name},
		Matches:      []string{},
		WithFiles:    false,
		AndCondition: false,
		Full:         false,
	}

	// NOTE: For now ignoring mask at this level.

	searcher := wagon.NewSearcherSimple(config)
	defer searcher.Close()

	res, err := searcher.SearchArtifacts(searchOpts)
	if err != nil {
		return "", fmt.Errorf("Error on resolve category for name %s: %s",
			name, err.Error())
	}

	if len(*res) == 0 {
		return "", fmt.Errorf("No matching packages found with name %s.", name)
	}

	// Convert artifacts to a map
	artsPack := &artifact.ArtifactsPack{
		Artifacts: *res,
	}
	artsMap := artsPack.ToMap()
	res = nil

	if len(artsMap.Artifacts) > 1 {

		errmsg := fmt.Sprintf("Multiple matches with name %s:\n", name)
		// Build packages matches list
		for k, _ := range artsMap.Artifacts {
			errmsg += "   - " + k + "\n"
		}

		return "", errors.New(errmsg)
	} else {
		ans = artsPack.Artifacts[0].GetPackage().GetCategory()
	}
	artsPack = nil
	artsMap = nil

	return ans, nil
}

func ParsePackageStr(config *cfg.LuetConfig, p string) (*pkg.DefaultPackage, error) {
	ver := ">=0"
	cat := ""
	name := ""
	cond := _gentoo.PackageCond(_gentoo.PkgCondInvalid)
	var err error

	if strings.Contains(p, "@") || !strings.Contains(p, "/") {
		packageinfo := strings.Split(p, "@")
		if len(packageinfo) > 1 {
			ver = packageinfo[1]
		}
		cat, name, cond = helpers.PackageResolveSplit(packageinfo[0])
		if cond != _gentoo.PkgCondInvalid && ver != "" {
			ver = cond.String() + ver
		}

		if cat == "" && config != nil {
			// POST: searching on enabled repo the package matching
			//       the name.
			cat, err = resolveCategory(config, name)
			if err != nil {
				return nil, err
			}
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

		ver = helpers.GentooVersion(gp)
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

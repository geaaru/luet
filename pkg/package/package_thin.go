/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package pkg

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"sort"
	"strings"

	"github.com/geaaru/luet/pkg/helpers/tools"
	gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

// PackageThin is a thin representation
// of a package. Normally is used to
// sort packages.
type PackageThin struct {
	Name      string         `yaml:"name" json:"name"`
	Category  string         `yaml:"category" json:"category"`
	Version   string         `yaml:"version" json:"version"`
	Requires  []*PackageThin `yaml:"requires,omitempty" json:"requires,omitempty"`
	Conflicts []*PackageThin `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
	Provides  []*PackageThin `yaml:"provides,omitempty" json:"provides,omitempty"`
}

func NewPackageThin(name, cat, version string,
	requires, conflicts []*PackageThin) *PackageThin {
	return &PackageThin{
		Name:      name,
		Category:  cat,
		Version:   version,
		Requires:  requires,
		Conflicts: conflicts,
		Provides:  []*PackageThin{},
	}
}

func (p *PackageThin) PackageName() string {
	if p.Category != "" && p.Name != "" {
		return fmt.Sprintf("%s/%s", p.Category, p.Name)
	} else if p.Category != "" {
		return p.Category
	} else {
		return p.Name
	}
}

func (p *PackageThin) ToGentooPackage() (*gentoo.GentooPackage, error) {

	var cond gentoo.PackageCond

	if strings.HasPrefix(p.Version, ">=") {
		cond = gentoo.PkgCondGreaterEqual
	} else if strings.HasPrefix(p.Version, "<=") {
		cond = gentoo.PkgCondLessEqual
	} else if strings.HasPrefix(p.Version, "!=") {
		cond = gentoo.PkgCondNot
	} else if strings.HasPrefix(p.Version, "=") {
		cond = gentoo.PkgCondEqual
	} else if strings.HasPrefix(p.Version, ">") {
		cond = gentoo.PkgCondGreater
	} else if strings.HasPrefix(p.Version, "<") {
		cond = gentoo.PkgCondLess
	}

	ans, err := gentoo.ParsePackageStr(
		fmt.Sprintf("%s-%s",
			p.PackageName(),
			strings.Trim(p.GetVersion(), "><=!"),
		),
	)

	if err != nil {
		return nil, err
	}

	ans.Condition = cond

	return ans, nil
}

func (p *PackageThin) GetVersion() string           { return p.Version }
func (p *PackageThin) GetCategory() string          { return p.Category }
func (p *PackageThin) GetName() string              { return p.Name }
func (p *PackageThin) GetProvides() []*PackageThin  { return p.Provides }
func (p *PackageThin) GetRequires() []*PackageThin  { return p.Requires }
func (p *PackageThin) GetConflicts() []*PackageThin { return p.Conflicts }

func (p *PackageThin) HasConflicts() bool {
	return tools.Ternary(p.Conflicts != nil, len(p.Conflicts) > 0, false)
}

func (p *PackageThin) HasRequires() bool {
	return tools.Ternary(p.Requires != nil, len(p.Requires) > 0, false)
}

func (p *PackageThin) HasProvides() bool {
	return tools.Ternary(p.Provides != nil, len(p.Provides) > 0, false)
}

func (p *PackageThin) RequirePackage(m *PackageThin) bool {
	for _, r := range p.Requires {
		if r.AtomMatches(m) {
			return true
		}
	}

	return false
}

func (p *PackageThin) AtomMatches(m *PackageThin) bool {
	if p.GetName() == m.GetName() && p.GetCategory() == m.GetCategory() {
		return true
	}
	return false
}

func (p *PackageThin) GenerateHash() string {
	var pmd5 hash.Hash = md5.New()

	b, _ := json.Marshal(p)

	pmd5.Write(b)

	var h []byte = pmd5.Sum(nil)

	return hex.EncodeToString(h)
}

func (p *PackageThin) HumanReadableString() string {
	return fmt.Sprintf("%s/%s-%s", p.Category, p.Name, p.Version)
}

func (p *PackageThin) String() string {
	return p.HumanReadableString()
}

func (p *PackageThin) BreakCyclesOnRequires(stack *[]*PackageThin) {
	newRequires := []*PackageThin{}

	for _, r := range p.Requires {
		present := false
		for _, pr := range *stack {
			if r.AtomMatches(pr) {
				present = true
				break
			}
		}

		if !present {
			newRequires = append(newRequires, r)
		}
	}

	p.Requires = newRequires
}

func PackageThinIsInList(p *PackageThin, list *[]*PackageThin) bool {
	ans := false

	for _, pp := range *list {
		if p.AtomMatches(pp) {
			ans = true
			break
		}
	}

	return ans
}

// Sort packages to have at the begin packages with
// zero or less requires and at the end the packages
// with more requires. If the number of requires are
// equal then it uses the PackageName() for sorting.
func SortPackageThinList4Requires(list *[]*PackageThin) {
	arr := *list

	sortPkgThin := func(i, j int) bool {
		pi := arr[i]
		pj := arr[j]
		ireq := pi.HasRequires()
		jreq := pj.HasRequires()

		if ireq && jreq {
			if len(pi.Requires) == len(pj.Requires) {
				return pi.PackageName() < pj.PackageName()
			}
			return len(pi.Requires) < len(pj.Requires)
		} else if !ireq && !jreq {
			return pi.PackageName() < pj.PackageName()
		} else if !ireq {
			return true
		}
		return false
	}

	sort.Slice(arr[:], sortPkgThin)
}

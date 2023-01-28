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

	"github.com/geaaru/luet/pkg/helpers/tools"
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

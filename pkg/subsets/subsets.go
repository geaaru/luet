// Copyright © 2021 Daniele Rondina <geaaru@sabayonlinux.org>
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

package subsets

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"

	"github.com/mudler/luet/pkg/helpers"

	"gopkg.in/yaml.v2"

	. "github.com/mudler/luet/pkg/config"
	. "github.com/mudler/luet/pkg/logger"
)

func NewLuetSubsetsConfig() *LuetSubsetsConfig {
	return &LuetSubsetsConfig{
		Enabled: []string{},
	}
}

func NewLuetSubsetsDefinition() *LuetSubsetsDefinition {
	return &LuetSubsetsDefinition{
		Definitions: make(map[string]*LuetSubsetDefinition, 0),
	}
}

func LoadSubsetsConfig(c *LuetConfig) error {
	var regexRepo = regexp.MustCompile(`.yml$|.yaml$`)
	var err error
	rootfs := ""

	// Respect the rootfs param on read repositories
	if !c.ConfigFromHost {
		rootfs, err = c.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	for _, sdir := range c.SubsetsConfDir {
		sdir = filepath.Join(rootfs, sdir)

		Debug("Parsing Subsets Configs Directory", sdir, "...")

		files, err := ioutil.ReadDir(sdir)
		if err != nil {
			Debug("Skip dir", sdir, ":", err.Error())
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !regexRepo.MatchString(file.Name()) {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			content, err := ioutil.ReadFile(path.Join(sdir, file.Name()))
			if err != nil {
				Warning("On read file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			sc, err := NewSubsetsConfig(content)
			if err != nil {
				Warning("On parser file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if len(sc.Enabled) == 0 {
				Warning("Invalid subset config ", file.Name())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			for _, e := range sc.Enabled {
				if !helpers.Contains(c.Subsets.Enabled, e) {
					c.Subsets.Enabled = append(c.Subsets.Enabled, e)
				}
			}

		}
	}

	return nil
}

func NewSubsetsConfig(data []byte) (*LuetSubsetsConfig, error) {
	ans := NewLuetSubsetsConfig()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func LoadSubsetsDef(data []byte) (*LuetSubsetsDefinition, error) {
	ans := NewLuetSubsetsDefinition()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func LoadSubsetsDefintions(c *LuetConfig) error {
	var regexRepo = regexp.MustCompile(`.yml$|.yaml$`)
	var err error
	rootfs := ""

	// Respect the rootfs param on read repositories
	if !c.ConfigFromHost {
		rootfs, err = c.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	for _, sdir := range c.SubsetsDefDir {
		sdir = filepath.Join(rootfs, sdir)

		Debug("Parsing Subsets Defintions Directory", sdir, "...")

		files, err := ioutil.ReadDir(sdir)
		if err != nil {
			Debug("Skip dir", sdir, ":", err.Error())
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !regexRepo.MatchString(file.Name()) {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			content, err := ioutil.ReadFile(path.Join(sdir, file.Name()))
			if err != nil {
				Warning("On read file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			r, err := LoadSubsetsDef(content)
			if err != nil {
				Warning("On parser file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if len(r.Definitions) == 0 {
				Warning("Invalid subsets defintion file", file.Name())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			mergeSubsetsDefinition(c, r)
		}
	}

	return nil
}

func mergeSubsetsDefinition(c *LuetConfig, s *LuetSubsetsDefinition) {
	for k, v := range s.Definitions {

		if len(v.Packages) == 0 && len(v.Categories) == 0 {
			// NOTE: override existing rules with last definition.
			c.SubsetsDefinitions.Definitions[k] = v
			continue
		}

		if len(v.Packages) > 0 {
			for _, p := range v.Packages {
				if _, ok := c.SubsetsPkgsDefMap[p]; ok {
					if _, ok2 := c.SubsetsPkgsDefMap[p].Definitions[v.Name]; ok2 {
						c.SubsetsPkgsDefMap[p].Definitions[v.Name].Rules =
							append(c.SubsetsPkgsDefMap[p].Definitions[v.Name].Rules, v.Rules...)
					} else {
						c.SubsetsPkgsDefMap[p].Definitions[v.Name] = s.Definitions[k]
					}
				} else {
					c.SubsetsPkgsDefMap[p] = &LuetSubsetsDefinition{
						Definitions: make(map[string]*LuetSubsetDefinition, 0),
					}
					c.SubsetsPkgsDefMap[p].Definitions[v.Name] = s.Definitions[k]
				}
			}
		}

		if len(v.Categories) > 0 {
			for _, cn := range v.Categories {
				if _, ok := c.SubsetsCatDefMap[cn]; ok {
					if _, ok2 := c.SubsetsCatDefMap[cn].Definitions[v.Name]; ok2 {

						c.SubsetsCatDefMap[cn].Definitions[v.Name].Rules =
							append(c.SubsetsCatDefMap[cn].Definitions[v.Name].Rules,
								v.Rules...)
					} else {
						c.SubsetsCatDefMap[cn].Definitions[v.Name] = s.Definitions[k]
					}
				} else {
					c.SubsetsCatDefMap[cn] = &LuetSubsetsDefinition{
						Definitions: make(map[string]*LuetSubsetDefinition, 0),
					}
					c.SubsetsCatDefMap[cn].Definitions[v.Name] = s.Definitions[k]
				}
			}
		}
	}
}

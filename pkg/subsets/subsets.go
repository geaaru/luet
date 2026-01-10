/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package subsets

import (
	"os"
	"path/filepath"
	"regexp"

	. "github.com/macaroni-os/anise/pkg/config"
	"github.com/macaroni-os/anise/pkg/helpers"
	. "github.com/macaroni-os/anise/pkg/logger"

	"gopkg.in/yaml.v3"
)

func LoadSubsetsConfig(c *AniseConfig) error {
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

		files, err := os.ReadDir(sdir)
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

			sc, err := NewSubsetsConfigFromFile(filepath.Join(sdir, file.Name()))
			if err != nil {
				Warning(err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if len(sc.Enabled) == 0 {
				// On using luet subsets disable we could have no subsets defined.
				// just ignoring the file.
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

func LoadSubsetsDef(data []byte) (*AniseSubsetsDefinition, error) {
	ans := NewAniseSubsetsDefinition()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func LoadSubsetsDefintions(c *AniseConfig) error {
	var regexRepo = regexp.MustCompile(`.yml$|.yaml$`)
	var err error
	rootfs := ""

	if c.SubsetsDefinitions == nil {
		c.SubsetsDefinitions = NewAniseSubsetsDefinition()
	}

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

		files, err := os.ReadDir(sdir)
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

			content, err := os.ReadFile(filepath.Join(sdir, file.Name()))
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

func mergeSubsetsDefinition(c *AniseConfig, s *AniseSubsetsDefinition) {
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
					c.SubsetsPkgsDefMap[p] = &AniseSubsetsDefinition{
						Definitions: make(map[string]*AniseSubsetDefinition, 0),
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
					c.SubsetsCatDefMap[cn] = &AniseSubsetsDefinition{
						Definitions: make(map[string]*AniseSubsetDefinition, 0),
					}
					c.SubsetsCatDefMap[cn].Definitions[v.Name] = s.Definitions[k]
				}
			}
		}
	}
}

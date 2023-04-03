/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package config

import (
	"fmt"
	"os"
	"path/filepath"

	helpers "github.com/geaaru/luet/pkg/helpers/file"

	"gopkg.in/yaml.v3"
)

func (c *LuetSubsetsConfig) HasSubset(s string) bool {
	ans := false
	for _, e := range c.Enabled {
		if e == s {
			ans = true
			break
		}
	}

	return ans
}

func (c *LuetSubsetsConfig) AddSubset(s string) {
	c.Enabled = append(c.Enabled, s)
}

func (c *LuetSubsetsConfig) DelSubset(s string) {
	enabled := []string{}
	for _, e := range c.Enabled {
		if e != s {
			enabled = append(enabled, e)
		}
	}
	c.Enabled = enabled
}

func (c *LuetSubsetsConfig) Write(f string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf(
			"Error on marshal subsets config: %s", err.Error())
	}

	basedir := filepath.Dir(f)
	if !helpers.Exists(basedir) {
		err := os.MkdirAll(basedir, os.ModePerm)
		if err != nil {
			return fmt.Errorf(
				"Error on create directory %s: %s", basedir,
				err.Error())
		}
	}

	err = os.WriteFile(f, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf(
			"Error on write file %s: %s", f, err.Error())
	}

	return nil
}

func NewLuetSubsetsConfig() *LuetSubsetsConfig {
	return &LuetSubsetsConfig{
		Enabled: []string{},
	}
}

func NewSubsetsConfig(data []byte) (*LuetSubsetsConfig, error) {
	ans := NewLuetSubsetsConfig()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}

	return ans, nil
}

func NewLuetSubsetsDefinition() *LuetSubsetsDefinition {
	return &LuetSubsetsDefinition{
		Definitions: make(map[string]*LuetSubsetDefinition, 0),
	}
}

func NewSubsetsConfigFromFile(f string) (*LuetSubsetsConfig, error) {
	fname := filepath.Base(f)

	content, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("On read file %s: %s", fname, err.Error())
	}

	ans, err := NewSubsetsConfig(content)
	if err != nil {
		return nil, fmt.Errorf("On parse file %s: %s", fname, err.Error())
	}

	return ans, nil
}

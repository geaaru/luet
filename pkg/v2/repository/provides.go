/*
Copyright © 2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"os"

	pkg "github.com/geaaru/luet/pkg/package"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

type WagonProvides struct {
	Provides map[string][]*pkg.DefaultPackage `yaml:"provides,omitempty" json:"provides,omitempty"`
}

func NewWagonProvides() *WagonProvides {
	return &WagonProvides{
		Provides: make(map[string][]*pkg.DefaultPackage, 0),
	}
}

func (wp *WagonProvides) Add(provname string, p *pkg.DefaultPackage) {
	if val, ok := wp.Provides[provname]; ok {
		wp.Provides[provname] = append(val, p)
	} else {
		wp.Provides[provname] = []*pkg.DefaultPackage{p}
	}
}

func (wp *WagonProvides) WriteProvidesYAML(dst string) error {

	// TODO: Using writer/reader without loading all bytes in memory.
	data, err := yaml.Marshal(wp)
	if err != nil {
		return errors.Wrap(err, "While marshalling for provides YAML")
	}

	err = os.WriteFile(dst, data, 0664)
	if err != nil {
		return errors.Wrap(err, "While writing provides YAML")
	}

	return nil
}

func (wp *WagonProvides) Load(f string) error {

	data, err := os.ReadFile(f)
	if err != nil {
		return errors.Wrap(err, "Error on reading file "+f)
	}

	err = yaml.Unmarshal(data, wp)
	if err != nil {
		return err
	}

	return nil
}

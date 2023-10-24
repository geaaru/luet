/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	pkg "github.com/geaaru/luet/pkg/package"
)

type CollectionRender struct {
	Packages []map[string]interface{} `json:"packages" yaml:"packages"`
}

func ReadCollectionFile(cFile string) (*pkg.Collection, error) {
	data, err := os.ReadFile(cFile)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error on read file %s: %s",
				cFile, err.Error()))
	}

	return NewCollection(&data)
}

func NewCollection(data *[]byte) (*pkg.Collection, error) {
	ans := &pkg.Collection{
		Packages: []pkg.DefaultPackage{},
	}

	if err := yaml.Unmarshal(*data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}

func ReadCollectionFileLoad(cFile string) (*CollectionRender, error) {
	data, err := os.ReadFile(cFile)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error on read file %s: %s",
				cFile, err.Error()))
	}

	return NewCollectionRender(&data)
}

func NewCollectionRender(data *[]byte) (*CollectionRender, error) {
	ans := &CollectionRender{
		Packages: []map[string]interface{}{},
	}

	if err := yaml.Unmarshal(*data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}

func (c *CollectionRender) GetPackageValues(p *pkg.DefaultPackage) (*map[string]interface{}, error) {
	gpSelector, err := p.ToGentooPackage()
	if err != nil {
		return nil, err
	}

	for idx, _ := range c.Packages {
		name, _ := c.Packages[idx]["name"].(string)
		cat, _ := c.Packages[idx]["category"].(string)
		version, _ := c.Packages[idx]["version"].(string)

		cPkg := pkg.NewPackageWithCatThin(cat, name, version)

		if p.AtomMatches(cPkg) {
			gp, _ := cPkg.ToGentooPackage()
			admitted, err := gpSelector.Admit(gp)
			if err != nil {
				return nil, err
			}

			if admitted {
				return &c.Packages[idx], nil
			}
		}
	}

	return nil, nil
}

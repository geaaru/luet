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

func ReadCollectionFile(cFile string) (*pkg.Collection, error) {
	ans := &pkg.Collection{
		Packages: []pkg.DefaultPackage{},
	}

	data, err := os.ReadFile(cFile)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error on read file %s: %s",
				cFile, err.Error()))
	}

	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}

	return ans, nil
}

/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"errors"
	"fmt"
	"os"

	pkg "github.com/geaaru/luet/pkg/package"
)

func ReadDefinitionFile(defFile string) (*pkg.DefaultPackage, error) {
	data, err := os.ReadFile(defFile)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error on read file %s: %s",
				defFile, err.Error()))
	}

	ans, err := pkg.NewDefaultPackageFromYaml(data)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Error on parse file %s: %s",
				defFile, err.Error()))
	}

	return ans, nil
}

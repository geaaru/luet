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
	spectooling "github.com/geaaru/luet/pkg/spectooling"
	"gopkg.in/yaml.v3"
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

func ReadDefinitionFileAsMap(defFile string) (*map[string]interface{}, error) {
	data, err := os.ReadFile(defFile)
	if err != nil {
		return nil, fmt.Errorf(
			"Error on read file %s: %s",
			defFile, err.Error())
	}

	ans := make(map[string]interface{}, 0)
	err = yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, fmt.Errorf(
			"Error on unmarshal data of file %s: %s",
			defFile, err.Error())
	}

	return &ans, nil
}

func WriteDefinitionFile(p pkg.Package, definitionFilePath string) error {
	data, err := spectooling.NewDefaultPackageSanitized(p).Yaml()
	if err != nil {
		return err
	}
	err = os.WriteFile(definitionFilePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/v2/compiler/types/specs"
	render "github.com/geaaru/luet/pkg/v2/render"

	"gopkg.in/yaml.v3"
)

// Render the build file of a package with definition.yaml file
func ReadBuildFile(buildFile, definitionFile string,
	engine *render.RenderEngine,
	overrideValues map[string]interface{},
) (*specs.CompilationSpecLoad, error) {

	pkgEngine := engine.CloneWithoutValues()

	if filepath.Base(definitionFile) == "definition.yaml" {
		err := pkgEngine.LoadValues([]string{definitionFile})
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Reading build file from collection without a selector")
	}

	buildSpecRendered, err := pkgEngine.RenderFile(buildFile, overrideValues)
	if err != nil {
		return nil, err
	}

	loadSpec := specs.NewComplationSpecLoad()
	err = yaml.Unmarshal([]byte(buildSpecRendered), &loadSpec)
	if err != nil {
		return nil, err
	}

	defPkg, err := ReadDefinitionFile(definitionFile)
	if err != nil {
		return nil, err
	}

	ans := loadSpec

	// NOTE: merging runtime requires, provides, conflicts only
	//       if the compiler specs are related to a virtual package
	if loadSpec.IsVirtual() {
		// If the requires aren't available we will use the runtime deps
		if len(loadSpec.GetRequires()) == 0 && len(defPkg.GetRequires()) != 0 {
			ans.Requires(defPkg.GetRequires())
		}

		if len(loadSpec.GetConflicts()) == 0 && len(defPkg.GetConflicts()) != 0 {
			ans.Conflicts(defPkg.GetConflicts())
		}

		if len(loadSpec.GetProvides()) == 0 && len(defPkg.GetProvides()) != 0 {
			ans.SetProvides(defPkg.GetProvides())
		}
	}

	return ans, nil
}

// Render the build file of a package with definition.yaml file
func ReadBuildFileFromCollection(buildFile, cFile string,
	engine *render.RenderEngine,
	atom *pkg.DefaultPackage,
	overrideValues map[string]interface{},
) (*specs.CompilationSpecLoad, error) {

	pkgEngine := engine.CloneWithoutValues()

	if filepath.Base(cFile) == "definition.yaml" {
		return nil, fmt.Errorf(
			"trying to render build file of a collection but definition.yaml available for package %s",
			atom.PackageName())
	}

	// Read data an
	data, err := os.ReadFile(cFile)
	if err != nil {
		return nil, fmt.Errorf(
			"Error on read file %s: %s",
			cFile, err.Error())
	}

	cRender, err := NewCollectionRender(&data)
	if err != nil {
		return nil, err
	}

	values, err := cRender.GetPackageValues(atom)
	if err != nil {
		return nil, err
	}

	pkgEngine.Values = *values
	cRender = nil

	collection, err := NewCollection(&data)

	defPkg, err := collection.GetPackage(
		atom.PackageName(), atom.Version)

	buildSpecRendered, err := pkgEngine.RenderFile(buildFile, overrideValues)
	if err != nil {
		return nil, err
	}

	loadSpec := specs.NewComplationSpecLoad()
	err = yaml.Unmarshal([]byte(buildSpecRendered), &loadSpec)
	if err != nil {
		return nil, err
	}

	ans := loadSpec

	// NOTE: merging runtime requires, provides, conflicts only
	//       if the compiler specs are related to a virtual package
	if loadSpec.IsVirtual() {
		// If the requires aren't available we will use the runtime deps
		if len(loadSpec.GetRequires()) == 0 && len(defPkg.GetRequires()) != 0 {
			ans.Requires(defPkg.GetRequires())
		}

		if len(loadSpec.GetConflicts()) == 0 && len(defPkg.GetConflicts()) != 0 {
			ans.Conflicts(defPkg.GetConflicts())
		}

		if len(loadSpec.GetProvides()) == 0 && len(defPkg.GetProvides()) != 0 {
			ans.SetProvides(defPkg.GetProvides())
		}
	}

	return ans, nil
}

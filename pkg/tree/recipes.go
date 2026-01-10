/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

// Recipe is a builder imeplementation.

// It reads a Tree and spit it in human readable form (YAML), called recipe,
// It also loads a tree (recipe) from a YAML (to a db, e.g. BoltDB), allowing to query it
// with the solver, using the package object.
package tree

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	pkg "github.com/macaroni-os/anise/pkg/package"
	spectooling "github.com/macaroni-os/anise/pkg/spectooling"

	"github.com/pkg/errors"
)

func NewGeneralRecipe(db pkg.PackageDatabase) Builder { return &Recipe{Database: db} }

// Recipe is the "general" reciper for Trees
type Recipe struct {
	SourcePath []string
	Database   pkg.PackageDatabase
}

func WriteDefinitionFile(p pkg.Package, definitionFilePath string) error {
	data, err := spectooling.NewDefaultPackageSanitized(p).Yaml()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(definitionFilePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *Recipe) Save(path string) error {
	for _, p := range r.Database.World() {
		dir := filepath.Join(path, p.GetCategory(), p.GetName(), p.GetVersion())
		os.MkdirAll(dir, os.ModePerm)

		err := WriteDefinitionFile(p, filepath.Join(dir, pkg.PackageDefinitionFile))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Recipe) Load(path string) error {

	// tmpfile, err := ioutil.TempFile("", "luet")
	// if err != nil {
	// 	return err
	// }
	if !fileHelper.Exists(path) {
		return errors.New(fmt.Sprintf(
			"Path %s doesn't exit.", path,
		))
	}

	r.SourcePath = append(r.SourcePath, path)

	if r.Database == nil {
		r.Database = pkg.NewInMemoryDatabase(false)
	}

	//r.Tree().SetPackageSet(pkg.NewBoltDatabase(tmpfile.Name()))
	// TODO: Handle cleaning after? Cleanup implemented in GetPackageSet().Clean()

	// the function that handles each file or dir
	var ff = func(currentpath string, info os.FileInfo, err error) error {

		if info.Name() != pkg.PackageDefinitionFile && info.Name() != pkg.PackageCollectionFile {
			return nil // Skip with no errors
		}

		dat, err := ioutil.ReadFile(currentpath)
		if err != nil {
			return errors.Wrap(err, "Error reading file "+currentpath)
		}

		switch info.Name() {
		case pkg.PackageDefinitionFile:
			pack, err := pkg.DefaultPackageFromYaml(dat)
			if err != nil {
				return errors.Wrap(err, "Error reading yaml "+currentpath)
			}

			// Path is set only internally when tree is loaded from disk
			pack.SetPath(filepath.Dir(currentpath))
			_, err = r.Database.CreatePackage(&pack)
			if err != nil {
				return errors.Wrap(err, "Error creating package "+pack.GetName())
			}
		case pkg.PackageCollectionFile:
			packs, err := pkg.DefaultPackagesFromYAML(dat)
			if err != nil {
				return errors.Wrap(err, "Error reading yaml "+currentpath)
			}
			for _, p := range packs {
				// Path is set only internally when tree is loaded from disk
				p.SetPath(filepath.Dir(currentpath))
				_, err = r.Database.CreatePackage(&p)
				if err != nil {
					return errors.Wrap(err, "Error creating package "+p.GetName())
				}
			}

		}

		return nil
	}

	err := filepath.Walk(path, ff)
	if err != nil {
		return err
	}
	return nil
}

func (r *Recipe) GetDatabase() pkg.PackageDatabase   { return r.Database }
func (r *Recipe) WithDatabase(d pkg.PackageDatabase) { r.Database = d }
func (r *Recipe) GetSourcePath() []string            { return r.SourcePath }

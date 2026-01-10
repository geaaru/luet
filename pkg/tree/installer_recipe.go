/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

// InstallerRecipe is a builder imeplementation.

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

	"github.com/pkg/errors"
)

const (
	FinalizerFile = "finalize.yaml"
)

func NewInstallerRecipe(db pkg.PackageDatabase) Builder {
	return &InstallerRecipe{Database: db}
}

// InstallerRecipe is the "general" reciper for Trees
type InstallerRecipe struct {
	SourcePath []string
	Database   pkg.PackageDatabase
}

func (r *InstallerRecipe) Save(path string) error {

	for _, p := range r.Database.World() {

		dir := filepath.Join(path, p.GetCategory(), p.GetName(), p.GetVersion())
		os.MkdirAll(dir, os.ModePerm)
		data, err := p.Yaml()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, pkg.PackageDefinitionFile), data, 0644)
		if err != nil {
			return err
		}
		// Instead of rdeps, have a different tree for build deps.
		finalizerPath := p.Rel(FinalizerFile)
		if fileHelper.Exists(finalizerPath) { // copy finalizer file from the source tree
			fileHelper.CopyFile(finalizerPath, filepath.Join(dir, FinalizerFile))
		}

	}
	return nil
}

func (r *InstallerRecipe) Load(path string) error {

	if !fileHelper.Exists(path) {
		return errors.New(fmt.Sprintf(
			"Path %s doesn't exit.", path,
		))
	}

	r.SourcePath = append(r.SourcePath, path)

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

func (r *InstallerRecipe) GetDatabase() pkg.PackageDatabase   { return r.Database }
func (r *InstallerRecipe) WithDatabase(d pkg.PackageDatabase) { r.Database = d }
func (r *InstallerRecipe) GetSourcePath() []string            { return r.SourcePath }

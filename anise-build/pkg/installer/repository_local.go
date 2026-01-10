/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	artifact "github.com/macaroni-os/anise/pkg/compiler/types/artifact"
	. "github.com/macaroni-os/anise/pkg/logger"
	pkg "github.com/macaroni-os/anise/pkg/package"

	"github.com/pkg/errors"
)

type localRepositoryGenerator struct{}

func (l *localRepositoryGenerator) Initialize(path string, db pkg.PackageDatabase) ([]*artifact.PackageArtifact, error) {
	return buildPackageIndex(path, db)
}

func buildPackageIndex(path string, db pkg.PackageDatabase) ([]*artifact.PackageArtifact, error) {

	var art []*artifact.PackageArtifact
	var ff = func(currentpath string, info os.FileInfo, err error) error {
		if err != nil {
			Debug("Failed walking", err.Error())
			return err
		}

		if !strings.HasSuffix(info.Name(), ".metadata.yaml") {
			return nil // Skip with no errors
		}

		dat, err := ioutil.ReadFile(currentpath)
		if err != nil {
			return errors.Wrap(err, "Error reading file "+currentpath)
		}

		a, err := artifact.NewPackageArtifactFromYaml(dat)
		if err != nil {
			return errors.Wrap(err, "Error reading yaml "+currentpath)
		} else if a.CompileSpec == nil {
			return errors.New(fmt.Sprintf("Unexpected metadata for artefact %v", a))
		}

		// We want to include packages that are ONLY referenced in the tree.
		// the ones which aren't should be deleted. (TODO: by another cli command?)
		if _, notfound := db.FindPackage(a.CompileSpec.GetPackage()); notfound != nil {
			Debug(fmt.Sprintf("Package %s not found in tree. Ignoring it.",
				a.CompileSpec.GetPackage().HumanReadableString()))
			return nil
		}

		art = append(art, a)

		return nil
	}

	err := filepath.Walk(path, ff)
	if err != nil {
		return nil, err

	}
	return art, nil
}

// Generate creates a Local luet repository
func (*localRepositoryGenerator) Generate(r *LuetSystemRepository, dst string, resetRevision bool) error {
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}
	r.LastUpdate = strconv.FormatInt(time.Now().Unix(), 10)

	repospec := filepath.Join(dst, REPOSITORY_SPECFILE)
	// Increment the internal revision version by reading the one which is already available (if any)
	if err := r.BumpRevision(repospec, resetRevision); err != nil {
		return err
	}

	Info(fmt.Sprintf(
		"Repository %s: creating revision %d and last update %s...",
		r.Name, r.Revision, r.LastUpdate,
	))

	if _, err := r.AddTree(r.GetTree(), dst, REPOFILE_TREE_KEY, NewDefaultTreeRepositoryFile()); err != nil {
		return errors.Wrap(err, "error met while adding runtime tree to repository")
	}

	if _, err := r.AddTree(r.BuildTree, dst, REPOFILE_COMPILER_TREE_KEY, NewDefaultCompilerTreeRepositoryFile()); err != nil {
		return errors.Wrap(err, "error met while adding compiler tree to repository")
	}

	if _, err := r.AddMetadata(repospec, dst); err != nil {
		return errors.Wrap(err, "failed adding Metadata file to repository")
	}

	return nil
}

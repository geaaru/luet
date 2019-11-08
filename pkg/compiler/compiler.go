// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package compiler

import (
	"errors"
	"io/ioutil"

	"github.com/mudler/luet/pkg/helpers"
	pkg "github.com/mudler/luet/pkg/package"
	"github.com/mudler/luet/pkg/tree"
)

const BuildFile = "build.yaml"

type LuetCompiler struct {
	*tree.CompilerRecipe
	Backend CompilerBackend
}

func NewLuetCompiler(backend CompilerBackend, t pkg.Tree) Compiler {
	// The CompilerRecipe will gives us a tree with only build deps listed.
	return &LuetCompiler{
		Backend: backend,
		CompilerRecipe: &tree.CompilerRecipe{
			tree.Recipe{PackageTree: t},
		},
	}
}

func (cs *LuetCompiler) Compile(p CompilationSpec) (Artifact, error) {

	// - If image is not set, we read a base_image. Then we will build one image from it to kick-off our build based
	// on how we compute the resolvable tree.
	// This means to recursively build all the build-images needed to reach that tree part.
	// - We later on compute an hash used to identify the image, so each similar deptree keeps the same build image.
	// - If image is set we just generate a plain dockerfile

	// Treat last case (easier) first. The image is provided and we just compute a plain dockerfile with the images listed as above

	if p.GetImage() != "" {
		p.SetSeedImage(p.GetImage()) // In this case, we ignore the build deps as we suppose that the image has them - otherwise we recompose the tree with a solver,
		// and we build all the images first.

		// First we create the builder image
		p.WriteBuildImageDefinition(p.Rel(p.GetPackage().GetFingerPrint() + "-builder.dockerfile"))
		builderOpts := CompilerBackendOptions{
			ImageName:      "luet/" + p.GetPackage().GetFingerPrint() + "-builder",
			SourcePath:     p.GetOutputPath(),
			DockerFileName: p.Rel(p.GetPackage().GetFingerPrint() + "-builder.dockerfile"),
			Destination:    p.Rel(p.GetPackage().GetFingerPrint() + "-builder.rootfs.tar"),
		}
		err := cs.Backend.BuildImage(builderOpts)
		if err != nil {
			return nil, err
		}

		// Then we write the step image, which uses the builder one
		p.WriteStepImageDefinition("luet/"+p.GetPackage().GetFingerPrint()+"-builder", p.Rel(p.GetPackage().GetFingerPrint()+".dockerfile"))
		runnerOpts := CompilerBackendOptions{
			ImageName:      "luet/" + p.GetPackage().GetFingerPrint(),
			SourcePath:     p.GetOutputPath(),
			DockerFileName: p.Rel(p.GetPackage().GetFingerPrint() + ".dockerfile"),
			Destination:    p.Rel(p.GetPackage().GetFingerPrint() + ".rootfs.tar"),
		}
		err = cs.Backend.ImageDefinitionToTar(runnerOpts)
		if err != nil {
			return nil, err
		}

		// TODO: Handle caching and optionally do not remove things
		err = cs.Backend.RemoveImage(builderOpts)
		if err != nil {
			return nil, err
		}

		// TODO: Delta should be the artifact
		return NewPackageArtifact(p.Rel(p.GetPackage().GetFingerPrint() + ".rootfs.tar")), nil
	}

	return nil, errors.New("Not implemented yet")
}

func (cs *LuetCompiler) FromPackage(p pkg.Package) (CompilationSpec, error) {

	pack, err := cs.Tree().GetPackageSet().FindPackage(p)
	if err != nil {
		return nil, err
	}
	buildFile := pack.Rel(BuildFile)
	if !helpers.Exists(buildFile) {
		return nil, errors.New("No build file present for " + p.GetFingerPrint())
	}

	dat, err := ioutil.ReadFile(buildFile)
	if err != nil {
		return nil, err
	}
	return NewLuetCompilationSpec(dat, p)
}

func (cs *LuetCompiler) GetBackend() CompilerBackend {
	return cs.Backend
}

func (cs *LuetCompiler) SetBackend(b CompilerBackend) {
	cs.Backend = b
}

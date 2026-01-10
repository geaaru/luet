/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package compilerspec

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	options "github.com/macaroni-os/anise/pkg/compiler/types/options"
	"github.com/mitchellh/hashstructure/v2"

	"github.com/ghodss/yaml"
	pkg "github.com/macaroni-os/anise/pkg/package"
	"github.com/macaroni-os/anise/pkg/solver"
	"github.com/otiai10/copy"
	dirhash "golang.org/x/mod/sumdb/dirhash"
)

type AniseCompilationspecs []AniseCompilationSpec

func NewAniseCompilationspecs(s ...*AniseCompilationSpec) *AniseCompilationspecs {
	all := AniseCompilationspecs{}

	for _, spec := range s {
		all.Add(spec)
	}
	return &all
}

func (specs AniseCompilationspecs) Len() int {
	return len(specs)
}

func (specs *AniseCompilationspecs) Remove(s *AniseCompilationspecs) *AniseCompilationspecs {
	newSpecs := AniseCompilationspecs{}
SPECS:
	for _, spec := range specs.All() {
		for _, target := range s.All() {
			if target.GetPackage().Matches(spec.GetPackage()) {
				continue SPECS
			}
		}
		newSpecs.Add(spec)
	}
	return &newSpecs
}

func (specs *AniseCompilationspecs) Add(s *AniseCompilationSpec) {
	*specs = append(*specs, *s)
}

func (specs *AniseCompilationspecs) All() []*AniseCompilationSpec {
	var cspecs []*AniseCompilationSpec
	for i, _ := range *specs {
		f := (*specs)[i]
		cspecs = append(cspecs, &f)
	}

	return cspecs
}

func (specs *AniseCompilationspecs) Unique() *AniseCompilationspecs {
	newSpecs := AniseCompilationspecs{}
	seen := map[string]bool{}

	for i, _ := range *specs {
		j := (*specs)[i]
		_, ok := seen[j.GetPackage().GetFingerPrint()]
		if !ok {
			seen[j.GetPackage().GetFingerPrint()] = true
			newSpecs = append(newSpecs, j)
		}
	}
	return &newSpecs
}

type CopyField struct {
	Package     *pkg.DefaultPackage `json:"package" yaml:"package"`
	Image       string              `json:"image" yaml:"image"`
	Source      string              `json:"source" yaml:"source"`
	Destination string              `json:"destination" yaml:"destination"`
}

type AniseCompilationSpec struct {
	Steps           []string                  `json:"steps" yaml:"steps"` // Are run inside a container and the result layer diff is saved
	Env             []string                  `json:"env" yaml:"env"`
	Prelude         []string                  `json:"prelude" yaml:"prelude"` // Are run inside the image which will be our builder
	Image           string                    `json:"image" yaml:"image"`
	Seed            string                    `json:"seed" yaml:"seed"`
	Package         *pkg.DefaultPackage       `json:"package" yaml:"package"`
	SourceAssertion solver.PackagesAssertions `json:"-" yaml:"-"`
	PackageDir      string                    `json:"package_dir" yaml:"package_dir"`

	Retrieve []string `json:"retrieve" yaml:"retrieve"`

	OutputPath string   `json:"-" yaml:"-"` // Where the build processfiles go
	Unpack     bool     `json:"unpack" yaml:"unpack"`
	Includes   []string `json:"includes" yaml:"includes"`
	Excludes   []string `json:"excludes" yaml:"excludes"`

	BuildOptions *options.Compiler `json:"build_options" yaml:"build_options"`

	Copy []CopyField `json:"copy" yaml:"copy"`

	RequiresFinalImages bool `json:"requires_final_images" yaml:"requires_final_images"`
}

// Signature is a portion of the spec that yields a signature for the hash
type Signature struct {
	Image               string
	Steps               []string
	PackageDir          string
	Prelude             []string
	Seed                string
	Env                 []string
	Retrieve            []string
	Unpack              bool
	Includes            []string
	Excludes            []string
	Copy                []CopyField
	Requires            pkg.DefaultPackages
	RequiresFinalImages bool
}

func (cs *AniseCompilationSpec) signature() Signature {
	return Signature{
		Image:               cs.Image,
		Steps:               cs.Steps,
		PackageDir:          cs.PackageDir,
		Prelude:             cs.Prelude,
		Seed:                cs.Seed,
		Env:                 cs.Env,
		Retrieve:            cs.Retrieve,
		Unpack:              cs.Unpack,
		Includes:            cs.Includes,
		Excludes:            cs.Excludes,
		Copy:                cs.Copy,
		Requires:            cs.Package.GetRequires(),
		RequiresFinalImages: cs.RequiresFinalImages,
	}
}

func NewAniseCompilationSpec(b []byte, p pkg.Package) (*AniseCompilationSpec, error) {
	var spec AniseCompilationSpec
	var packageDefinition pkg.DefaultPackage
	err := yaml.Unmarshal(b, &spec)
	if err != nil {
		return &spec, err
	}
	err = yaml.Unmarshal(b, &packageDefinition)
	if err != nil {
		return &spec, err
	}

	// Update requires/conflict/provides
	// When we have been passed a bytes slice, parse it as a package
	// and updates requires/conflicts/provides.
	// This is required in order to allow manipulation of such fields with templating
	copy := *p.(*pkg.DefaultPackage)
	spec.Package = &copy
	if len(packageDefinition.GetRequires()) != 0 {
		spec.Package.Requires(packageDefinition.GetRequires())
	}
	if len(packageDefinition.GetConflicts()) != 0 {
		spec.Package.Conflicts(packageDefinition.GetConflicts())
	}
	if len(packageDefinition.GetProvides()) != 0 {
		spec.Package.SetProvides(packageDefinition.GetProvides())
	}
	return &spec, nil
}
func (cs *AniseCompilationSpec) GetSourceAssertion() solver.PackagesAssertions {
	return cs.SourceAssertion
}

func (cs *AniseCompilationSpec) SetBuildOptions(b options.Compiler) {
	cs.BuildOptions = &b
}

func (cs *AniseCompilationSpec) IsValid() (bool, error) {

	if !cs.IsVirtual() {
		if cs.Image == "" {
			if len(cs.Package.GetRequires()) == 0 && len(cs.Copy) == 0 {
				return false,
					errors.New("No requires, image or layer to join found")
			}
		}
	}

	return true, nil
}

func (cs *AniseCompilationSpec) SetSourceAssertion(as solver.PackagesAssertions) {
	cs.SourceAssertion = as
}
func (cs *AniseCompilationSpec) GetPackage() pkg.Package {
	return cs.Package
}

func (cs *AniseCompilationSpec) GetPackageDir() string {
	return cs.PackageDir
}

func (cs *AniseCompilationSpec) SetPackageDir(s string) {
	cs.PackageDir = s
}

func (cs *AniseCompilationSpec) BuildSteps() []string {
	return cs.Steps
}

func (cs *AniseCompilationSpec) ImageUnpack() bool {
	return cs.Unpack
}

func (cs *AniseCompilationSpec) GetPreBuildSteps() []string {
	return cs.Prelude
}

func (cs *AniseCompilationSpec) GetIncludes() []string {
	return cs.Includes
}

func (cs *AniseCompilationSpec) GetExcludes() []string {
	return cs.Excludes
}

func (cs *AniseCompilationSpec) GetRetrieve() []string {
	return cs.Retrieve
}

// IsVirtual returns true if the spec is virtual.
// A spec is virtual if the package is empty, and it has no image source to unpack from.
func (cs *AniseCompilationSpec) IsVirtual() bool {
	return cs.EmptyPackage() && !cs.HasImageSource()
}

func (cs *AniseCompilationSpec) GetSeedImage() string {
	return cs.Seed
}

func (cs *AniseCompilationSpec) GetImage() string {
	return cs.Image
}

func (cs *AniseCompilationSpec) GetOutputPath() string {
	return cs.OutputPath
}

func (p *AniseCompilationSpec) Rel(s string) string {
	return filepath.Join(p.GetOutputPath(), s)
}

func (cs *AniseCompilationSpec) SetImage(s string) {
	cs.Image = s
}

func (cs *AniseCompilationSpec) SetOutputPath(s string) {
	cs.OutputPath = s
}

func (cs *AniseCompilationSpec) SetSeedImage(s string) {
	cs.Seed = s
}

func (cs *AniseCompilationSpec) EmptyPackage() bool {
	return len(cs.BuildSteps()) == 0 && !cs.UnpackedPackage()
}

func (cs *AniseCompilationSpec) UnpackedPackage() bool {
	// If package_dir was specified in the spec, we want to treat the content of the directory
	// as the root of our archive.  ImageUnpack is implied to be true. override it
	unpack := cs.ImageUnpack()
	if cs.GetPackageDir() != "" {
		unpack = true
	}
	return unpack
}

// HasImageSource returns true when the compilation spec has an image source.
// a compilation spec has an image source when it depends on other packages or have a source image
// explictly supplied
func (cs *AniseCompilationSpec) HasImageSource() bool {
	return (cs.Package != nil && len(cs.GetPackage().GetRequires()) != 0) || cs.GetImage() != "" || (cs.RequiresFinalImages && len(cs.Package.GetRequires()) != 0)
}

func (cs *AniseCompilationSpec) Hash() (string, error) {
	// build a signature, we want to be part of the hash only the fields that are relevant for build purposes
	signature := cs.signature()
	h, err := hashstructure.Hash(signature, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	sum, err := dirhash.HashDir(cs.Package.Path, "", dirhash.DefaultHash)
	if err != nil {
		return fmt.Sprint(h), err
	}
	return fmt.Sprint(h, sum), err
}

func (cs *AniseCompilationSpec) CopyRetrieves(dest string) error {
	var err error
	if len(cs.Retrieve) > 0 {
		for _, s := range cs.Retrieve {
			matches, err := filepath.Glob(cs.Rel(s))

			if err != nil {
				continue
			}

			for _, m := range matches {
				err = copy.Copy(m, filepath.Join(dest, filepath.Base(m)))
			}
		}
	}
	return err
}

func (cs *AniseCompilationSpec) genDockerfile(image string, steps []string) string {
	spec := `
FROM ` + image + `
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=` + cs.Package.GetName() + `
ENV PACKAGE_VERSION=` + cs.Package.GetVersion() + `
ENV PACKAGE_CATEGORY=` + cs.Package.GetCategory()

	if len(cs.Retrieve) > 0 {
		for _, s := range cs.Retrieve {
			//var file string
			// if helpers.IsValidUrl(s) {
			// 	file = s
			// } else {
			// 	file = cs.Rel(s)
			// }
			spec = spec + `
ADD ` + s + ` /anisebuild/`
		}
	}

	for _, c := range cs.Copy {
		if c.Image != "" {
			copyLine := fmt.Sprintf("\nCOPY --from=%s %s %s\n", c.Image, c.Source, c.Destination)
			spec = spec + copyLine
		}
	}

	for _, s := range cs.Env {
		spec = spec + `
ENV ` + s
	}

	for _, s := range steps {
		spec = spec + `
RUN ` + s
	}
	return spec
}

// RenderBuildImage renders the dockerfile of the image used as a pre-build step
func (cs *AniseCompilationSpec) RenderBuildImage() (string, error) {
	return cs.genDockerfile(cs.GetSeedImage(), cs.GetPreBuildSteps()), nil

}

// RenderStepImage renders the dockerfile used for the image used for building the package
func (cs *AniseCompilationSpec) RenderStepImage(image string) (string, error) {
	return cs.genDockerfile(image, cs.BuildSteps()), nil
}

func (cs *AniseCompilationSpec) WriteBuildImageDefinition(path string) error {
	data, err := cs.RenderBuildImage()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(data), 0644)
}

func (cs *AniseCompilationSpec) WriteStepImageDefinition(fromimage, path string) error {
	data, err := cs.RenderStepImage(fromimage)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(data), 0644)
}

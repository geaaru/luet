/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package compilerspec_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	options "github.com/macaroni-os/anise/pkg/compiler/types/options"
	compilerspec "github.com/macaroni-os/anise/pkg/compiler/types/spec"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"

	. "github.com/macaroni-os/anise/pkg/compiler"
	pkg "github.com/macaroni-os/anise/pkg/package"
	"github.com/macaroni-os/anise/pkg/tree"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spec", func() {
	Context("Anise specs", func() {
		It("Allows normal operations", func() {
			testSpec := &compilerspec.AniseCompilationSpec{Package: &pkg.DefaultPackage{Name: "foo", Category: "a", Version: "0"}}
			testSpec2 := &compilerspec.AniseCompilationSpec{Package: &pkg.DefaultPackage{Name: "bar", Category: "a", Version: "0"}}
			testSpec3 := &compilerspec.AniseCompilationSpec{Package: &pkg.DefaultPackage{Name: "baz", Category: "a", Version: "0"}}
			testSpec4 := &compilerspec.AniseCompilationSpec{Package: &pkg.DefaultPackage{Name: "foo", Category: "a", Version: "0"}}

			specs := compilerspec.NewAniseCompilationspecs(testSpec, testSpec2)
			Expect(specs.Len()).To(Equal(2))
			Expect(specs.All()).To(Equal([]*compilerspec.AniseCompilationSpec{testSpec, testSpec2}))
			specs.Add(testSpec3)
			Expect(specs.All()).To(Equal([]*compilerspec.AniseCompilationSpec{testSpec, testSpec2, testSpec3}))
			specs.Add(testSpec4)
			Expect(specs.All()).To(Equal([]*compilerspec.AniseCompilationSpec{testSpec, testSpec2, testSpec3, testSpec4}))
			newSpec := specs.Unique()
			Expect(newSpec.All()).To(Equal([]*compilerspec.AniseCompilationSpec{testSpec, testSpec2, testSpec3}))

			newSpec2 := specs.Remove(compilerspec.NewAniseCompilationspecs(testSpec, testSpec2))
			Expect(newSpec2.All()).To(Equal([]*compilerspec.AniseCompilationSpec{testSpec3}))

		})
		Context("virtuals", func() {
			When("is empty", func() {
				It("is virtual", func() {
					spec := &compilerspec.AniseCompilationSpec{}
					Expect(spec.IsVirtual()).To(BeTrue())
				})
			})
			When("has defined steps", func() {
				It("is not a virtual", func() {
					spec := &compilerspec.AniseCompilationSpec{Steps: []string{"foo"}}
					Expect(spec.IsVirtual()).To(BeFalse())
				})
			})
			When("has defined image", func() {
				It("is not a virtual", func() {
					spec := &compilerspec.AniseCompilationSpec{Image: "foo"}
					Expect(spec.IsVirtual()).To(BeFalse())
				})
			})
		})
	})

	Context("Image hashing", func() {
		It("is stable", func() {
			spec1 := &compilerspec.AniseCompilationSpec{
				Image:        "foo",
				BuildOptions: &options.Compiler{BuildValues: []map[string]interface{}{{"foo": "bar", "baz": true}}},

				Package: &pkg.DefaultPackage{
					Name:     "foo",
					Category: "Bar",
					Labels: map[string]string{
						"foo": "bar",
						"baz": "foo",
					},
				},
			}
			spec2 := &compilerspec.AniseCompilationSpec{
				Image:        "foo",
				BuildOptions: &options.Compiler{BuildValues: []map[string]interface{}{{"foo": "bar", "baz": true}}},
				Package: &pkg.DefaultPackage{
					Name:     "foo",
					Category: "Bar",
					Labels: map[string]string{
						"foo": "bar",
						"baz": "foo",
					},
				},
			}
			spec3 := &compilerspec.AniseCompilationSpec{
				Image: "foo",
				Steps: []string{"foo"},
				Package: &pkg.DefaultPackage{
					Name:     "foo",
					Category: "Bar",
					Labels: map[string]string{
						"foo": "bar",
						"baz": "foo",
					},
				},
			}
			hash, err := spec1.Hash()
			Expect(err).ToNot(HaveOccurred())

			hash2, err := spec2.Hash()
			Expect(err).ToNot(HaveOccurred())

			hash3, err := spec3.Hash()
			Expect(err).ToNot(HaveOccurred())

			Expect(hash).To(Equal(hash2))
			hashagain, err := spec2.Hash()
			Expect(err).ToNot(HaveOccurred())
			Expect(hash).ToNot(Equal(hash3))
			Expect(hash).To(Equal(hashagain))
		})
	})

	Context("Simple package build definition", func() {
		It("Loads it correctly", func() {
			generalRecipe := tree.NewGeneralRecipe(pkg.NewInMemoryDatabase(false))

			err := generalRecipe.Load("../../../../tests/fixtures/buildtree")
			Expect(err).ToNot(HaveOccurred())

			Expect(len(generalRecipe.GetDatabase().GetPackages())).To(Equal(1))

			compiler := NewAniseCompiler(nil, generalRecipe.GetDatabase())
			lspec, err := compiler.FromPackage(&pkg.DefaultPackage{Name: "enman", Category: "app-admin", Version: "1.4.0"})
			Expect(err).ToNot(HaveOccurred())

			Expect(lspec.Steps).To(Equal([]string{"echo foo > /test", "echo bar > /test2"}))
			Expect(lspec.Image).To(Equal("anise/base"))
			Expect(lspec.Seed).To(Equal("alpine"))
			tmpdir, err := ioutil.TempDir("", "tree")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpdir) // clean up

			lspec.Env = []string{"test=1"}
			err = lspec.WriteBuildImageDefinition(filepath.Join(tmpdir, "Dockerfile"))
			Expect(err).ToNot(HaveOccurred())
			dockerfile, err := fileHelper.Read(filepath.Join(tmpdir, "Dockerfile"))
			Expect(err).ToNot(HaveOccurred())
			Expect(dockerfile).To(Equal(`
FROM alpine
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=enman
ENV PACKAGE_VERSION=1.4.0
ENV PACKAGE_CATEGORY=app-admin
ENV test=1`))

			err = lspec.WriteStepImageDefinition(lspec.Image, filepath.Join(tmpdir, "Dockerfile"))
			Expect(err).ToNot(HaveOccurred())
			dockerfile, err = fileHelper.Read(filepath.Join(tmpdir, "Dockerfile"))
			Expect(err).ToNot(HaveOccurred())
			Expect(dockerfile).To(Equal(`
FROM anise/base
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=enman
ENV PACKAGE_VERSION=1.4.0
ENV PACKAGE_CATEGORY=app-admin
ENV test=1
RUN echo foo > /test
RUN echo bar > /test2`))

		})

	})

	It("Renders retrieve and env fields", func() {
		generalRecipe := tree.NewGeneralRecipe(pkg.NewInMemoryDatabase(false))

		err := generalRecipe.Load("../../../../tests/fixtures/retrieve")
		Expect(err).ToNot(HaveOccurred())

		Expect(len(generalRecipe.GetDatabase().GetPackages())).To(Equal(1))

		compiler := NewAniseCompiler(nil, generalRecipe.GetDatabase())
		lspec, err := compiler.FromPackage(&pkg.DefaultPackage{Name: "a", Category: "test", Version: "1.0"})
		Expect(err).ToNot(HaveOccurred())

		Expect(lspec.Steps).To(Equal([]string{"echo foo > /test", "echo bar > /test2"}))
		Expect(lspec.Image).To(Equal("anise/base"))
		Expect(lspec.Seed).To(Equal("alpine"))
		tmpdir, err := ioutil.TempDir("", "tree")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(tmpdir) // clean up

		err = lspec.WriteBuildImageDefinition(filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())
		dockerfile, err := fileHelper.Read(filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())
		Expect(dockerfile).To(Equal(`
FROM alpine
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=a
ENV PACKAGE_VERSION=1.0
ENV PACKAGE_CATEGORY=test
ADD test /anisebuild/
ADD http://www.google.com /anisebuild/
ENV test=1`))

		lspec.SetOutputPath("/foo/bar")

		err = lspec.WriteBuildImageDefinition(filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())
		dockerfile, err = fileHelper.Read(filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())
		Expect(dockerfile).To(Equal(`
FROM alpine
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=a
ENV PACKAGE_VERSION=1.0
ENV PACKAGE_CATEGORY=test
ADD test /anisebuild/
ADD http://www.google.com /anisebuild/
ENV test=1`))

		err = lspec.WriteStepImageDefinition(lspec.Image, filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())
		dockerfile, err = fileHelper.Read(filepath.Join(tmpdir, "Dockerfile"))
		Expect(err).ToNot(HaveOccurred())

		Expect(dockerfile).To(Equal(`
FROM anise/base
COPY . /anisebuild
WORKDIR /anisebuild
ENV PACKAGE_NAME=a
ENV PACKAGE_VERSION=1.0
ENV PACKAGE_CATEGORY=test
ADD test /anisebuild/
ADD http://www.google.com /anisebuild/
ENV test=1
RUN echo foo > /test
RUN echo bar > /test2`))

	})
})

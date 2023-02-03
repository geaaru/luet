// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
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

package compiler_test

import (
	"fmt"

	. "github.com/geaaru/luet/pkg/compiler"
	sd "github.com/geaaru/luet/pkg/compiler/backend"
	"github.com/geaaru/luet/pkg/compiler/types/options"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/tree"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageHashTree", func() {
	generalRecipe := tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))
	compiler := NewLuetCompiler(
		sd.NewSimpleDockerBackend(),
		generalRecipe.GetDatabase(),
		options.Concurrency(2),
	)
	hashtree := NewHashTree(generalRecipe.GetDatabase())
	Context("Simple package definition", func() {
		BeforeEach(func() {
			generalRecipe = tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))
			err := generalRecipe.Load("../../tests/fixtures/buildable")
			Expect(err).ToNot(HaveOccurred())
			compiler = NewLuetCompiler(
				sd.NewSimpleDockerBackend(),
				generalRecipe.GetDatabase(),
				options.Concurrency(2),
			)
			hashtree = NewHashTree(generalRecipe.GetDatabase())

		})

		It("Calculates the hash correctly", func() {

			spec, err := compiler.FromPackage(
				&pkg.DefaultPackage{
					Name:     "b",
					Category: "test",
					Version:  "1.0",
				})
			Expect(err).ToNot(HaveOccurred())

			packageHash, err := hashtree.Query(compiler, spec)
			Expect(err).ToNot(HaveOccurred())
			fmt.Println("Package Target Hash BuildHash ", packageHash.Target.Hash.BuildHash)
			fmt.Println("Package Target PackageHash ", packageHash.Target.Hash.PackageHash)
			fmt.Println("Package Builder Image Hash", packageHash.BuilderImageHash)
			Expect(packageHash.Target.Hash.BuildHash).To(
				Equal("895697a8bb51b219b78ed081fa1b778801e81505bb03f56acafcf3c476620fc1"))
			Expect(packageHash.Target.Hash.PackageHash).To(
				Equal("2a6c3dc0dd7af2902fd8823a24402d89b2030cfbea6e63fe81afb34af8b1a005"))
			Expect(packageHash.BuilderImageHash).To(
				Equal("builder-3a28d240f505d69123735a567beaf80e"))
		})
	})

	//expectedPackageHash := "f3f42a7435293225e92a51da8416f90b7c0ccd5958cd5c72276c39ece408c01f"
	expectedPackageHash := "2110248439f3695b8ded9e96f85a186ab2ab42ca7ed1f3f06a0ec1a36c0b6281"

	Context("complex package definition", func() {
		BeforeEach(func() {
			generalRecipe = tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))

			err := generalRecipe.Load("../../tests/fixtures/upgrade_old_repo_revision")
			Expect(err).ToNot(HaveOccurred())
			compiler = NewLuetCompiler(sd.NewSimpleDockerBackend(), generalRecipe.GetDatabase(), options.Concurrency(2))
			hashtree = NewHashTree(generalRecipe.GetDatabase())

		})
		It("Calculates the hash correctly", func() {
			spec, err := compiler.FromPackage(&pkg.DefaultPackage{Name: "c", Category: "test", Version: "1.0"})
			Expect(err).ToNot(HaveOccurred())

			packageHash, err := hashtree.Query(compiler, spec)
			Expect(err).ToNot(HaveOccurred())

			fmt.Println("Package hash last dependency",
				packageHash.Dependencies[len(packageHash.Dependencies)-1].Hash.PackageHash,
				packageHash.Dependencies[len(packageHash.Dependencies)-1],
				packageHash.Target.Hash.PackageHash,
			)
			Expect(packageHash.Dependencies[len(packageHash.Dependencies)-1].Hash.PackageHash).To(Equal(expectedPackageHash))
			Expect(packageHash.SourceHash).To(Equal(expectedPackageHash))
			Expect(packageHash.BuilderImageHash).To(Equal(
				"builder-ad9a301b463fa7336fbaa51908e8e073",
			))
			//Expect(packageHash.BuilderImageHash).To(Equal("builder-977129605c0d7e974cc8a431a563cec1"))

			//Expect(packageHash.Target.Hash.BuildHash).To(Equal("79d7107d13d578b362e6a7bf10ec850efce26316405b8d732ce8f9e004d64281"))
			//Expect(packageHash.Target.Hash.PackageHash).To(Equal("9112e2c97bf8ca998c1df303a9ebc4957b685930c882e9aa556eab4507220079"))
			Expect(packageHash.Target.Hash.PackageHash).To(Equal("cfeeefe631ea3f857510f8af2326974de99fbf7cbec7835f29412be990a86519"))
			a := &pkg.DefaultPackage{Name: "a", Category: "test", Version: "1.1"}
			hash, err := packageHash.DependencyBuildImage(a)
			Expect(err).ToNot(HaveOccurred())

			Expect(hash).To(Equal("a1130e6bf3a93314e39ad59bf5753457714a2a013584b9499279a14515a20d76"))

			assertionA := packageHash.Dependencies.Search(a.GetFingerPrint())
			Expect(assertionA.Hash.PackageHash).To(Equal(expectedPackageHash))
			b := &pkg.DefaultPackage{Name: "b", Category: "test", Version: "1.0"}
			assertionB := packageHash.Dependencies.Search(b.GetFingerPrint())

			Expect(assertionB.Hash.PackageHash).To(Equal("a1130e6bf3a93314e39ad59bf5753457714a2a013584b9499279a14515a20d76"))
			hashB, err := packageHash.DependencyBuildImage(b)
			Expect(err).ToNot(HaveOccurred())

			Expect(hashB).To(Equal("a1e11506ac69a1f2049e1f5fa9e675e9d80c9b164bbce8f0d986e46283572dd5"))
		})
	})

	Context("complex package definition, with small change in build.yaml", func() {
		BeforeEach(func() {
			generalRecipe = tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))

			//Definition of A here is slightly changed in the steps build.yaml file (1 character only)
			err := generalRecipe.Load("../../tests/fixtures/upgrade_old_repo_revision_content_changed")
			Expect(err).ToNot(HaveOccurred())
			compiler = NewLuetCompiler(sd.NewSimpleDockerBackend(), generalRecipe.GetDatabase(), options.Concurrency(2))
			hashtree = NewHashTree(generalRecipe.GetDatabase())

		})
		It("Calculates the hash correctly", func() {
			spec, err := compiler.FromPackage(&pkg.DefaultPackage{Name: "c", Category: "test", Version: "1.0"})
			Expect(err).ToNot(HaveOccurred())

			packageHash, err := hashtree.Query(compiler, spec)
			Expect(err).ToNot(HaveOccurred())
			fmt.Println("Package hash ",
				packageHash.Dependencies[len(packageHash.Dependencies)-1].Hash.PackageHash,
				packageHash.Target.Hash.PackageHash)
			Expect(packageHash.Dependencies[len(packageHash.Dependencies)-1].Hash.PackageHash).ToNot(Equal(expectedPackageHash))
			sourceHash := "fcaba1ecac3525a42439f5f3aaf7be29e884890dc9a7679904d66fe26f1b5993"
			Expect(packageHash.Dependencies[len(packageHash.Dependencies)-1].Hash.PackageHash).To(Equal(sourceHash))
			Expect(packageHash.SourceHash).To(Equal(sourceHash))

			Expect(packageHash.SourceHash).ToNot(Equal(expectedPackageHash))

			Expect(packageHash.BuilderImageHash).To(Equal("builder-4b46a6376d4753fbcb5ebddb7b81d98c"))

			//Expect(packageHash.Target.Hash.BuildHash).To(Equal("79d7107d13d578b362e6a7bf10ec850efce26316405b8d732ce8f9e004d64281"))
			Expect(packageHash.Target.Hash.PackageHash).To(Equal("0a62e10e4c89f2a5246d7eeca2dfbba9eeade322fe9a6eb33298bd6867adb9d2"))
			a := &pkg.DefaultPackage{Name: "a", Category: "test", Version: "1.1"}
			hash, err := packageHash.DependencyBuildImage(a)
			Expect(err).ToNot(HaveOccurred())

			Expect(hash).To(Equal("b4b61939260263582da1dfa5289182a0a7570ef8658f3b01b1997fe5d8a95e49"))

			assertionA := packageHash.Dependencies.Search(a.GetFingerPrint())

			Expect(assertionA.Hash.PackageHash).To(Equal("fcaba1ecac3525a42439f5f3aaf7be29e884890dc9a7679904d66fe26f1b5993"))
			Expect(assertionA.Hash.PackageHash).ToNot(Equal(expectedPackageHash))

			b := &pkg.DefaultPackage{Name: "b", Category: "test", Version: "1.0"}
			assertionB := packageHash.Dependencies.Search(b.GetFingerPrint())

			Expect(assertionB.Hash.PackageHash).To(Equal("b4b61939260263582da1dfa5289182a0a7570ef8658f3b01b1997fe5d8a95e49"))
			hashB, err := packageHash.DependencyBuildImage(b)
			Expect(err).ToNot(HaveOccurred())

			Expect(hashB).To(Equal("fc6fdd4bd62d51fc06c2c22e8bc56543727a2340220972594e28c623ea3a9c6c"))
		})
	})

})

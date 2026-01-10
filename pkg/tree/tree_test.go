/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package tree_test

import (
	"io/ioutil"
	"os"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pkg "github.com/macaroni-os/anise/pkg/package"
	"github.com/macaroni-os/anise/pkg/solver"
	. "github.com/macaroni-os/anise/pkg/tree"
)

var _ = Describe("Tree", func() {

	Context("Simple solving with the fixture tree", func() {
		It("writes and reads back the same tree", func() {
			for index := 0; index < 300; index++ { // Just to make sure we don't have false positives
				db := pkg.NewInMemoryDatabase(false)
				generalRecipe := NewCompilerRecipe(db)
				tmpdir, err := ioutil.TempDir("", "package")
				Expect(err).ToNot(HaveOccurred())
				defer os.RemoveAll(tmpdir) // clean up

				err = generalRecipe.Load("../../tests/fixtures/buildableseed")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(generalRecipe.GetDatabase().World())).To(Equal(4))

				D, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "d", Category: "test", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())

				Expect(D.GetRequires()[0].GetName()).To(Equal("c"))
				CfromD, err := generalRecipe.GetDatabase().FindPackage(D.GetRequires()[0])
				Expect(err).ToNot(HaveOccurred())

				Expect(len(CfromD.GetRequires()) != 0).To(BeTrue())
				Expect(CfromD.GetRequires()[0].GetName()).To(Equal("b"))

				s := solver.NewSolver(solver.Options{Type: solver.SingleCoreSimple}, pkg.NewInMemoryDatabase(false), generalRecipe.GetDatabase(), db)
				pack, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "d", Category: "test", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())

				solution, err := s.Install([]pkg.Package{pack})
				Expect(err).ToNot(HaveOccurred())

				solution, err = solution.Order(generalRecipe.GetDatabase(), pack.GetFingerPrint())
				Expect(err).ToNot(HaveOccurred())

				Expect(solution[0].Package.GetName()).To(Equal("b"))
				Expect(solution[0].Value).To(BeTrue())

				Expect(solution[1].Package.GetName()).To(Equal("c"))
				Expect(solution[1].Value).To(BeTrue())

				Expect(solution[2].Package.GetName()).To(Equal("d"))
				Expect(solution[2].Value).To(BeTrue())
				Expect(len(solution)).To(Equal(3))

				newsolution := solution.Drop(&pkg.DefaultPackage{Name: "d", Category: "test", Version: "1.0"})
				Expect(len(newsolution)).To(Equal(2))

				Expect(newsolution[0].Package.GetName()).To(Equal("b"))
				Expect(newsolution[0].Value).To(BeTrue())

				Expect(newsolution[1].Package.GetName()).To(Equal("c"))
				Expect(newsolution[1].Value).To(BeTrue())

			}
		})
	})

	Context("Multiple trees", func() {
		It("Merges", func() {
			for index := 0; index < 300; index++ { // Just to make sure we don't have false positives
				db := pkg.NewInMemoryDatabase(false)
				generalRecipe := NewCompilerRecipe(db)
				tmpdir, err := ioutil.TempDir("", "package")
				Expect(err).ToNot(HaveOccurred())
				defer os.RemoveAll(tmpdir) // clean up

				err = generalRecipe.Load("../../tests/fixtures/buildableseed")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(generalRecipe.GetDatabase().World())).To(Equal(4))

				err = generalRecipe.Load("../../tests/fixtures/layers")
				Expect(err).ToNot(HaveOccurred())

				Expect(len(generalRecipe.GetDatabase().World())).To(Equal(6))

				extra, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "extra", Category: "layer", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())
				Expect(extra).ToNot(BeNil())

				D, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "d", Category: "test", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())

				Expect(D.GetRequires()[0].GetName()).To(Equal("c"))
				CfromD, err := generalRecipe.GetDatabase().FindPackage(D.GetRequires()[0])
				Expect(err).ToNot(HaveOccurred())

				Expect(len(CfromD.GetRequires()) != 0).To(BeTrue())
				Expect(CfromD.GetRequires()[0].GetName()).To(Equal("b"))

				s := solver.NewSolver(solver.Options{Type: solver.SingleCoreSimple}, pkg.NewInMemoryDatabase(false), generalRecipe.GetDatabase(), db)

				Dd, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "d", Category: "test", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())

				solution, err := s.Install([]pkg.Package{Dd})
				Expect(err).ToNot(HaveOccurred())

				solution, err = solution.Order(generalRecipe.GetDatabase(), Dd.GetFingerPrint())
				Expect(err).ToNot(HaveOccurred())
				pack, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "a", Category: "test", Version: "1.0"})
				Expect(err).ToNot(HaveOccurred())

				base, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "base", Category: "layer", Version: "0.2"})
				Expect(err).ToNot(HaveOccurred())
				Expect(solution).ToNot(ContainElement(solver.PackageAssert{Package: pack.(*pkg.DefaultPackage), Value: true}))
				Expect(solution).To(ContainElement(solver.PackageAssert{Package: D.(*pkg.DefaultPackage), Value: true}))
				Expect(solution).ToNot(ContainElement(solver.PackageAssert{Package: extra.(*pkg.DefaultPackage), Value: true}))
				Expect(solution).ToNot(ContainElement(solver.PackageAssert{Package: base.(*pkg.DefaultPackage), Value: true}))
				Expect(len(solution)).To(Equal(3))
			}
		})
	})

	Context("Simple tree with labels", func() {
		It("Read tree with labels", func() {
			db := pkg.NewInMemoryDatabase(false)
			generalRecipe := NewCompilerRecipe(db)

			err := generalRecipe.Load("../../tests/fixtures/labels")
			Expect(err).ToNot(HaveOccurred())

			Expect(len(generalRecipe.GetDatabase().World())).To(Equal(1))
			pack, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "pkgA", Category: "test", Version: "0.1"})
			Expect(err).ToNot(HaveOccurred())

			Expect(pack.HasLabel("label1")).To(Equal(true))
			Expect(pack.HasLabel("label3")).To(Equal(false))
		})
	})

	Context("Simple tree with annotations", func() {
		It("Read tree with annotations", func() {
			db := pkg.NewInMemoryDatabase(false)
			generalRecipe := NewCompilerRecipe(db)
			r := regexp.MustCompile("^label")

			err := generalRecipe.Load("../../tests/fixtures/annotations")
			Expect(err).ToNot(HaveOccurred())

			Expect(len(generalRecipe.GetDatabase().World())).To(Equal(1))
			pack, err := generalRecipe.GetDatabase().FindPackage(&pkg.DefaultPackage{Name: "pkgA", Category: "test", Version: "0.1"})
			Expect(err).ToNot(HaveOccurred())

			Expect(pack.HasAnnotation("label1")).To(Equal(true))
			Expect(pack.HasAnnotation("label3")).To(Equal(false))
			Expect(pack.MatchAnnotation(r)).To(Equal(true))
		})
	})

})

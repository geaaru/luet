/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package installer_test

import (
	. "github.com/macaroni-os/anise/anise-build/pkg/installer"
	pkg "github.com/macaroni-os/anise/pkg/package"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("System", func() {
	Context("Files", func() {
		var s *System
		var db pkg.PackageDatabase
		var a, b *pkg.DefaultPackage

		BeforeEach(func() {
			db = pkg.NewInMemoryDatabase(false)
			s = &System{Database: db}

			a = &pkg.DefaultPackage{Name: "test", Version: "1", Category: "t"}

			db.CreatePackage(a)
			db.SetPackageFiles(&pkg.PackageFile{PackageFingerprint: a.GetFingerPrint(), Files: []string{"foo", "f"}})

			b = &pkg.DefaultPackage{Name: "test2", Version: "1", Category: "t"}

			db.CreatePackage(b)
			db.SetPackageFiles(&pkg.PackageFile{PackageFingerprint: b.GetFingerPrint(), Files: []string{"barz", "f"}})
		})

		It("detects when are already shipped by other packages", func() {
			r, p, err := s.ExistsPackageFile("foo")
			Expect(r).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			Expect(p).To(Equal(a))
			r, p, err = s.ExistsPackageFile("baz")
			Expect(r).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
			Expect(p).To(BeNil())

			r, p, err = s.ExistsPackageFile("f")
			Expect(r).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			Expect(p).To(Equal(b))
			r, p, err = s.ExistsPackageFile("barz")
			Expect(r).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			Expect(p).To(Equal(b))
		})
	})
})

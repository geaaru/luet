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

package installer_test

import (

	//	. "github.com/geaaru/luet/pkg/installer"

	. "github.com/geaaru/luet/pkg/installer"
	pkg "github.com/geaaru/luet/pkg/package"

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

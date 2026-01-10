/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package config_test

import (
	config "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("Test config protect", func() {

		It("Protect1", func() {

			files := []string{
				"etc/foo/my.conf",
				"usr/bin/foo",
				"usr/share/doc/foo.md",
			}

			cp := config.NewConfigProtect("/etc")
			cp.Map(files)

			Expect(cp.Protected("etc/foo/my.conf")).To(BeTrue())
			Expect(cp.Protected("/etc/foo/my.conf")).To(BeTrue())
			Expect(cp.Protected("usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/share/doc/foo.md")).To(BeFalse())

			Expect(cp.GetProtectFiles(false)).To(Equal(
				[]string{
					"etc/foo/my.conf",
				},
			))

			Expect(cp.GetProtectFiles(true)).To(Equal(
				[]string{
					"/etc/foo/my.conf",
				},
			))
		})

		It("Protect2", func() {

			files := []string{
				"etc/foo/my.conf",
				"usr/bin/foo",
				"usr/share/doc/foo.md",
			}

			cp := config.NewConfigProtect("")
			cp.Map(files)

			Expect(cp.Protected("etc/foo/my.conf")).To(BeFalse())
			Expect(cp.Protected("/etc/foo/my.conf")).To(BeFalse())
			Expect(cp.Protected("usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/share/doc/foo.md")).To(BeFalse())

			Expect(cp.GetProtectFiles(false)).To(Equal(
				[]string{},
			))

			Expect(cp.GetProtectFiles(true)).To(Equal(
				[]string{},
			))
		})

		It("Protect3: Annotation dir without initial slash", func() {

			files := []string{
				"etc/foo/my.conf",
				"usr/bin/foo",
				"usr/share/doc/foo.md",
			}

			cp := config.NewConfigProtect("etc")
			cp.Map(files)

			Expect(cp.Protected("etc/foo/my.conf")).To(BeTrue())
			Expect(cp.Protected("/etc/foo/my.conf")).To(BeTrue())
			Expect(cp.Protected("usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/bin/foo")).To(BeFalse())
			Expect(cp.Protected("/usr/share/doc/foo.md")).To(BeFalse())

			Expect(cp.GetProtectFiles(false)).To(Equal(
				[]string{
					"etc/foo/my.conf",
				},
			))

			Expect(cp.GetProtectFiles(true)).To(Equal(
				[]string{
					"/etc/foo/my.conf",
				},
			))
		})

	})

})

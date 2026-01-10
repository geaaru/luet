/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd_helpers_test

import (
	"path/filepath"

	. "github.com/macaroni-os/anise/cmd/helpers"
	cfg "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI Helpers", func() {

	dbpath, _ := filepath.Abs("../../tests/repo-trees")
	// At the moment the wagon use global variable
	// to retrieve database Path.
	config := cfg.LuetCfg
	config.GetSystem().DatabasePath = dbpath
	repo := cfg.NewLuetRepository("mottainai-stable", "http",
		"Mottainai Stable Repo",
		[]string{"http://mydomain.it"},
		10, true, true)
	config.SystemRepositories = append(
		config.SystemRepositories, *repo)

	Context("Can parse package strings correctly", func() {
		It("accept single package names", func() {

			pack, err := ParsePackageStr(config, "foo")
			Expect(err).To(HaveOccurred())
			Expect(pack == nil).To(Equal(true))
			Expect(err.Error()).To(Equal("No matching packages found with name foo."))
		})

		It("accept single package names and resolve category", func() {

			pack, err := ParsePackageStr(config, "lxd-compose")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("lxd-compose"))
			Expect(pack.GetCategory()).To(Equal("app-emulation"))
			Expect(pack.GetVersion()).To(Equal(">=0"))
		})
		It("accept unversioned packages with category", func() {
			pack, err := ParsePackageStr(nil, "cat/foo")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal(">=0"))
		})
		It("accept versioned packages with category", func() {
			pack, err := ParsePackageStr(nil, "cat/foo@1.1")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal("1.1"))
		})
		It("accept versioned ranges with category", func() {
			pack, err := ParsePackageStr(nil, "cat/foo@>=1.1")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal(">=1.1"))
		})
		It("accept gentoo regex parsing without versions", func() {
			pack, err := ParsePackageStr(nil, "=cat/foo")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal(">=0"))
		})
		It("accept gentoo regex parsing with versions", func() {
			pack, err := ParsePackageStr(nil, "=cat/foo-1.2")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal("1.2"))
		})

		It("accept gentoo regex parsing with with condition", func() {
			pack, err := ParsePackageStr(nil, ">=cat/foo-1.2")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal(">=1.2"))
		})

		It("accept gentoo regex parsing with with condition2", func() {
			pack, err := ParsePackageStr(nil, "<cat/foo-1.2")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal("<1.2"))
		})

		It("accept gentoo regex parsing with with condition3", func() {
			pack, err := ParsePackageStr(nil, ">cat/foo-1.2")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal(">1.2"))
		})

		It("accept gentoo regex parsing with with condition4", func() {
			pack, err := ParsePackageStr(nil, "<=cat/foo-1.2")
			Expect(err).ToNot(HaveOccurred())
			Expect(pack.GetName()).To(Equal("foo"))
			Expect(pack.GetCategory()).To(Equal("cat"))
			Expect(pack.GetVersion()).To(Equal("<=1.2"))
		})
	})
})

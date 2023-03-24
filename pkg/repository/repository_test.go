/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package repository_test

import (
	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/repository"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	viper "github.com/spf13/viper"
)

var _ = Describe("Repository", func() {
	Context("Load Repository1", func() {
		cfg := NewLuetConfig(viper.New())
		cfg.RepositoriesConfDir = []string{
			"../../tests/fixtures/repos.conf.d",
		}
		err := LoadRepositories(cfg)

		It("Chec Load Repository 1", func() {
			Expect(err).Should(BeNil())
			Expect(len(cfg.SystemRepositories)).Should(Equal(2))
			Expect(cfg.SystemRepositories[0].Name).Should(Equal("test1"))
			Expect(cfg.SystemRepositories[0].Priority).Should(Equal(999))
			Expect(cfg.SystemRepositories[0].Type).Should(Equal("disk"))
			Expect(len(cfg.SystemRepositories[0].Urls)).Should(Equal(1))
			Expect(cfg.SystemRepositories[0].Urls[0]).Should(Equal("tests/repos/test1"))
		})

		It("Chec Load Repository 2", func() {
			Expect(err).Should(BeNil())
			Expect(len(cfg.SystemRepositories)).Should(Equal(2))
			Expect(cfg.SystemRepositories[1].Name).Should(Equal("test2"))
			Expect(cfg.SystemRepositories[1].Priority).Should(Equal(1000))
			Expect(cfg.SystemRepositories[1].Type).Should(Equal("disk"))
			Expect(len(cfg.SystemRepositories[1].Urls)).Should(Equal(1))
			Expect(cfg.SystemRepositories[1].Urls[0]).Should(Equal("tests/repos/test2"))
		})
	})
})

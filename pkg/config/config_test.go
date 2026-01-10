/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package config_test

import (
	"os"
	"path/filepath"
	"strings"

	config "github.com/macaroni-os/anise/pkg/config"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("Simple temporary directory creation", func() {

		It("Create Temporary directory", func() {
			// PRE: tmpdir_base contains default value.

			tmpDir, err := config.AniseCfg.GetSystem().TempDir("test1")
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.HasPrefix(tmpDir, filepath.Join(os.TempDir(), "tmpanise"))).To(BeTrue())
			Expect(fileHelper.Exists(tmpDir)).To(BeTrue())

			defer os.RemoveAll(tmpDir)
		})

		It("Create Temporary file", func() {
			// PRE: tmpdir_base contains default value.

			tmpFile, err := config.AniseCfg.GetSystem().TempFile("testfile1")
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.HasPrefix(tmpFile.Name(), filepath.Join(os.TempDir(), "tmpanise"))).To(BeTrue())
			Expect(fileHelper.Exists(tmpFile.Name())).To(BeTrue())

			defer os.Remove(tmpFile.Name())
		})

		It("Config1", func() {
			cfg := config.AniseCfg

			cfg.GetLogging().Color = false
			Expect(cfg.GetLogging().Color).To(BeFalse())
		})

	})

})

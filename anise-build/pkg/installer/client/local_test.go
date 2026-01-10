/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package client_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/macaroni-os/anise/anise-build/pkg/installer/client"
	"github.com/macaroni-os/anise/pkg/compiler/types/artifact"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Local client", func() {
	Context("With repository", func() {
		It("Downloads single files", func() {
			tmpdir, err := ioutil.TempDir("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpdir) // clean up

			// write the whole body at once
			err = ioutil.WriteFile(filepath.Join(tmpdir, "test.txt"), []byte(`test`), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			c := NewLocalClient(RepoData{Urls: []string{tmpdir}})
			path, err := c.DownloadFile("test.txt")
			Expect(err).ToNot(HaveOccurred())
			Expect(fileHelper.Read(path)).To(Equal("test"))
			os.RemoveAll(path)
		})

		It("Downloads artifacts", func() {
			tmpdir, err := ioutil.TempDir("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpdir) // clean up

			// write the whole body at once
			err = ioutil.WriteFile(filepath.Join(tmpdir, "test.txt"), []byte(`test`), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			c := NewLocalClient(RepoData{Urls: []string{tmpdir}})
			path, err := c.DownloadArtifact(&artifact.PackageArtifact{Path: "test.txt"})
			Expect(err).ToNot(HaveOccurred())
			Expect(fileHelper.Read(path.Path)).To(Equal("test"))
			os.RemoveAll(path.Path)
		})

	})
})

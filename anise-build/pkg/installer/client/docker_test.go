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
	compilerspec "github.com/macaroni-os/anise/pkg/compiler/types/spec"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	pkg "github.com/macaroni-os/anise/pkg/package"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This test expect that the repository defined in UNIT_TEST_DOCKER_IMAGE is in zstd format.
// the repository is built by the 01_simple_docker.sh integration test fileHelper.
// This test also require root. At the moment, unpacking docker images with 'img' requires root permission to
// mount/unmount layers.
var _ = Describe("Docker client", func() {
	Context("With repository", func() {
		repoImage := os.Getenv("UNIT_TEST_DOCKER_IMAGE")
		var repoURL []string
		var c *DockerClient
		BeforeEach(func() {
			if repoImage == "" {
				Skip("UNIT_TEST_DOCKER_IMAGE not specified")
			}
			repoURL = []string{repoImage}
			c = NewDockerClient(RepoData{Urls: repoURL})
		})

		It("Downloads single files", func() {
			f, err := c.DownloadFile("repository.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(fileHelper.Read(f)).To(ContainSubstring("Test Repo"))
			os.RemoveAll(f)
		})

		It("Downloads artifacts", func() {
			f, err := c.DownloadArtifact(&artifact.PackageArtifact{
				Path: "test.tar",
				CompileSpec: &compilerspec.LuetCompilationSpec{
					Package: &pkg.DefaultPackage{
						Name:     "c",
						Category: "test",
						Version:  "1.0",
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			tmpdir, err := ioutil.TempDir("", "test")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpdir) // clean up
			Expect(f.Unpack(tmpdir, false)).ToNot(HaveOccurred())
			Expect(fileHelper.Read(filepath.Join(tmpdir, "c"))).To(Equal("c\n"))
			Expect(fileHelper.Read(filepath.Join(tmpdir, "cd"))).To(Equal("c\n"))
			os.RemoveAll(f.Path)
		})
	})
})

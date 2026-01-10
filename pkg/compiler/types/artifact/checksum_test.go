/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package artifact_test

import (
	"io/ioutil"
	"os"

	. "github.com/macaroni-os/anise/pkg/compiler/types/artifact"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Checksum", func() {
	Context("Generation", func() {
		It("Compares successfully", func() {

			tmpdir, err := ioutil.TempDir("", "tree")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpdir) // clean up
			buildsum := Checksums{}
			definitionsum := Checksums{}
			definitionsum2 := Checksums{}

			Expect(len(buildsum)).To(Equal(0))
			Expect(len(definitionsum)).To(Equal(0))
			Expect(len(definitionsum2)).To(Equal(0))

			err = buildsum.Generate(NewPackageArtifact("../../../../tests/fixtures/layers/alpine/build.yaml"))
			Expect(err).ToNot(HaveOccurred())

			err = definitionsum.Generate(NewPackageArtifact("../../../../tests/fixtures/layers/alpine/definition.yaml"))
			Expect(err).ToNot(HaveOccurred())

			err = definitionsum2.Generate(NewPackageArtifact("../../../../tests/fixtures/layers/alpine/definition.yaml"))
			Expect(err).ToNot(HaveOccurred())

			Expect(len(buildsum)).To(Equal(1))
			Expect(len(definitionsum)).To(Equal(1))
			Expect(len(definitionsum2)).To(Equal(1))

			Expect(definitionsum.Compare(buildsum)).To(HaveOccurred())
			Expect(definitionsum.Compare(definitionsum2)).ToNot(HaveOccurred())
		})
	})

})

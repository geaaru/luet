/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package compiler_test

import (
	. "github.com/macaroni-os/anise/pkg/compiler"
	. "github.com/macaroni-os/anise/pkg/compiler/backend"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Docker image diffs", func() {
	var b CompilerBackend

	BeforeEach(func() {
		b = NewSimpleDockerBackend()
	})

	Context("Generate diffs from docker images", func() {
		It("Detect no changes", func() {
			opts := Options{
				ImageName: "alpine:latest",
			}
			err := b.DownloadImage(opts)
			Expect(err).ToNot(HaveOccurred())

			layers, err := GenerateChanges(b, opts, opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(layers)).To(Equal(1))
			Expect(len(layers[0].Diffs.Additions)).To(Equal(0))
			Expect(len(layers[0].Diffs.Changes)).To(Equal(0))
			Expect(len(layers[0].Diffs.Deletions)).To(Equal(0))
		})

		It("Detects additions and changed files", func() {
			err := b.DownloadImage(Options{
				ImageName: "quay.io/mocaccino/micro",
			})
			Expect(err).ToNot(HaveOccurred())
			err = b.DownloadImage(Options{
				ImageName: "quay.io/mocaccino/extra",
			})
			Expect(err).ToNot(HaveOccurred())

			layers, err := GenerateChanges(b, Options{
				ImageName: "quay.io/mocaccino/micro",
			}, Options{
				ImageName: "quay.io/mocaccino/extra",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(layers)).To(Equal(1))

			/*
				Expect(len(layers[0].Diffs.Changes) > 0).To(BeTrue())
				Expect(len(layers[0].Diffs.Changes[0].Name) > 0).To(BeTrue())
				Expect(layers[0].Diffs.Changes[0].Size > 0).To(BeTrue())

				Expect(len(layers[0].Diffs.Additions) > 0).To(BeTrue())
				Expect(len(layers[0].Diffs.Additions[0].Name) > 0).To(BeTrue())
				Expect(layers[0].Diffs.Additions[0].Size > 0).To(BeTrue())

				Expect(len(layers[0].Diffs.Deletions)).To(Equal(0))

			*/
		})
	})
})

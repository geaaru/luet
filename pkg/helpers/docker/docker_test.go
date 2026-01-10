/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package docker_test

import (
	"github.com/macaroni-os/anise/pkg/helpers/docker"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StripInvalidStringsFromImage", func() {
	Context("Image names", func() {
		It("strips invalid chars", func() {
			Expect(docker.StripInvalidStringsFromImage("foo+bar")).To(Equal("foo-bar"))
		})
	})
})

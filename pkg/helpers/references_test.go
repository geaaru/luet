/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package helpers_test

import (
	. "github.com/macaroni-os/anise/pkg/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	Context("StripRegistryFromImage", func() {
		It("Strips the domain name", func() {
			out := StripRegistryFromImage("valid.domain.org/base/image:tag")
			Expect(out).To(Equal("base/image:tag"))
		})
		It("Strips the domain name when port is included", func() {
			out := StripRegistryFromImage("valid.domain.org:5000/base/image:tag")
			Expect(out).To(Equal("base/image:tag"))
		})
		It("Does not strip the domain name", func() {
			out := StripRegistryFromImage("not-a-domain/base/image:tag")
			Expect(out).To(Equal("not-a-domain/base/image:tag"))
		})
		It("Does not strip the domain name on invalid domains", func() {
			out := StripRegistryFromImage("-invaliddomain.org/base/image:tag")
			Expect(out).To(Equal("-invaliddomain.org/base/image:tag"))
		})
	})
})

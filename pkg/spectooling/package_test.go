/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package spectooling_test

import (
	pkg "github.com/macaroni-os/anise/pkg/package"
	. "github.com/macaroni-os/anise/pkg/spectooling"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spec Tooling", func() {
	Context("Conversion1", func() {

		b := pkg.NewPackage("B", "1.0", []*pkg.DefaultPackage{}, []*pkg.DefaultPackage{})
		c := pkg.NewPackage("C", "1.0", []*pkg.DefaultPackage{}, []*pkg.DefaultPackage{})
		d := pkg.NewPackage("D", "1.0", []*pkg.DefaultPackage{}, []*pkg.DefaultPackage{})
		p1 := pkg.NewPackage("A", "1.0", []*pkg.DefaultPackage{b, c}, []*pkg.DefaultPackage{d})
		virtual := pkg.NewPackage("E", "1.0", []*pkg.DefaultPackage{}, []*pkg.DefaultPackage{})
		virtual.SetCategory("virtual")
		p1.Provides = []*pkg.DefaultPackage{virtual}
		p1.AddLabel("label1", "value1")
		p1.AddLabel("label2", "value2")
		p1.SetDescription("Package1")
		p1.SetCategory("cat1")
		p1.SetLicense("GPL")
		p1.AddURI("https://github.com/macaroni-os/anise")
		p1.AddUse("systemd")
		It("Convert pkg1", func() {
			res := NewDefaultPackageSanitized(p1)
			expected_res := &DefaultPackageSanitized{
				Name:     "A",
				Version:  "1.0",
				Category: "cat1",
				PackageRequires: []*DefaultPackageSanitized{
					&DefaultPackageSanitized{
						Name:    "B",
						Version: "1.0",
					},
					&DefaultPackageSanitized{
						Name:    "C",
						Version: "1.0",
					},
				},
				PackageConflicts: []*DefaultPackageSanitized{
					&DefaultPackageSanitized{
						Name:    "D",
						Version: "1.0",
					},
				},
				Provides: []*DefaultPackageSanitized{
					&DefaultPackageSanitized{
						Name:     "E",
						Category: "virtual",
						Version:  "1.0",
					},
				},
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
				Description: "Package1",
				License:     "GPL",
				Uri:         []string{"https://github.com/macaroni-os/anise"},
				UseFlags:    []string{"systemd"},
			}

			Expect(res).To(Equal(expected_res))
		})

	})
})

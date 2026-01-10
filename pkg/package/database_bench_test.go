/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package pkg_test

import (
	"io/ioutil"
	"os"
	"strconv"

	. "github.com/macaroni-os/anise/pkg/package"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database  benchmark", func() {

	Context("BoltDB", func() {

		a := NewPackage("A", ">=1.0", []*DefaultPackage{}, []*DefaultPackage{})

		tmpfile, _ := ioutil.TempFile(os.TempDir(), "tests")
		defer os.Remove(tmpfile.Name()) // clean up
		var db PackageSet

		BeforeEach(func() {

			tmpfile, _ = ioutil.TempFile(os.TempDir(), "tests")
			defer os.Remove(tmpfile.Name()) // clean up
			db = NewBoltDatabase(tmpfile.Name())
			if os.Getenv("BENCHMARK_TESTS") != "true" {
				Skip("BENCHMARK_TESTS not enabled")
			}
		})

		Measure("it should be fast in computing world from a 50000 dataset", func(b Benchmarker) {
			for i := 0; i < 50000; i++ {
				a = NewPackage("A"+strconv.Itoa(i), ">=1.0", []*DefaultPackage{}, []*DefaultPackage{})

				_, err := db.CreatePackage(a)
				Expect(err).ToNot(HaveOccurred())
			}
			runtime := b.Time("runtime", func() {
				packs := db.World()
				Expect(len(packs)).To(Equal(50000))
			})

			Ω(runtime.Seconds()).Should(BeNumerically("<", 30), "World() shouldn't take too long.")

		}, 1)

		Measure("it should be fast in computing world from a 100000 dataset", func(b Benchmarker) {
			for i := 0; i < 100000; i++ {
				a = NewPackage("A"+strconv.Itoa(i), ">=1.0", []*DefaultPackage{}, []*DefaultPackage{})

				_, err := db.CreatePackage(a)
				Expect(err).ToNot(HaveOccurred())
			}
			runtime := b.Time("runtime", func() {
				packs := db.World()
				Expect(len(packs)).To(Equal(100000))
			})

			Ω(runtime.Seconds()).Should(BeNumerically("<", 30), "World() shouldn't take too long.")

		}, 1)
	})

	Context("InMemory", func() {

		a := NewPackage("A", ">=1.0", []*DefaultPackage{}, []*DefaultPackage{})

		tmpfile, _ := ioutil.TempFile(os.TempDir(), "tests")
		defer os.Remove(tmpfile.Name()) // clean up
		var db PackageSet

		BeforeEach(func() {

			tmpfile, _ = ioutil.TempFile(os.TempDir(), "tests")
			defer os.Remove(tmpfile.Name()) // clean up
			db = NewInMemoryDatabase(false)
			if os.Getenv("BENCHMARK_TESTS") != "true" {
				Skip("BENCHMARK_TESTS not enabled")
			}
		})

		Measure("it should be fast in computing world from a 100000 dataset", func(b Benchmarker) {

			runtime := b.Time("runtime", func() {
				for i := 0; i < 100000; i++ {
					a = NewPackage("A"+strconv.Itoa(i), ">=1.0", []*DefaultPackage{}, []*DefaultPackage{})

					_, err := db.CreatePackage(a)
					Expect(err).ToNot(HaveOccurred())
				}
				packs := db.World()
				Expect(len(packs)).To(Equal(100000))
			})

			Ω(runtime.Seconds()).Should(BeNumerically("<", 10), "World() shouldn't take too long.")

		}, 2)
	})
})

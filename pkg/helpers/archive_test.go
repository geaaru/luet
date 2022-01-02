// Copyright © 2019-2020 Ettore Di Giacinto <mudler@gentoo.org>
//                       Daniele Rondina <geaaru@sabayonlinux.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package helpers_test

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
	fileHelper "github.com/mudler/luet/pkg/helpers/file"

	. "github.com/mudler/luet/pkg/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Code from moby/moby pkg/archive/archive_test
func prepareUntarSourceDirectory(numberOfFiles int, targetPath string, makeLinks bool) (int, error) {
	fileData := []byte("fooo")
	for n := 0; n < numberOfFiles; n++ {
		fileName := fmt.Sprintf("file-%d", n)
		if err := ioutil.WriteFile(filepath.Join(targetPath, fileName), fileData, 0700); err != nil {
			return 0, err
		}
		if makeLinks {
			if err := os.Link(filepath.Join(targetPath, fileName), filepath.Join(targetPath, fileName+"-link")); err != nil {
				return 0, err
			}
		}
	}
	totalSize := numberOfFiles * len(fileData)
	return totalSize, nil
}

func tarModifierWrapperFunc(path, dir string, header *tar.Header,
	content io.Reader, opts *tarf.TarFileOperation, t *tarf.TarFormers) error {

	// If the destination path already exists I rename target file name with postfix.
	var basePath string
	buffer := bytes.Buffer{}
	if content != nil {
		if _, err := buffer.ReadFrom(content); err != nil {
			return err
		}
	}

	if header != nil {

		switch header.Typeflag {
		case tar.TypeReg:
			basePath = filepath.Base(path)
		default:
			// Nothing to do. I return original reader
			return nil
		}

		if basePath == "file-0" {
			path = filepath.Join(filepath.Join(filepath.Dir(path), fmt.Sprintf("._cfg%04d_%s", 1, basePath)))
		}

		// else file not present
	}

	info := header.FileInfo()
	opts.Skip = true
	// Write the file
	err := t.CreateFile(dir, path, info.Mode(), bytes.NewReader(buffer.Bytes()), header)
	if err != nil {
		return err
	}
	meta := tarf_specs.NewFileMeta(header)
	err = t.SetFileProps(filepath.Join(dir, path), &meta, false)
	return nil
}

var _ = Describe("Helpers Archive", func() {
	Context("Untar Protect", func() {

		It("Detect existing and not-existing files", func() {

			archiveSourceDir, err := ioutil.TempDir("", "archive-source")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(archiveSourceDir)

			_, err = prepareUntarSourceDirectory(10, archiveSourceDir, false)
			Expect(err).ToNot(HaveOccurred())

			targetDir, err := ioutil.TempDir("", "archive-target")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(targetDir)

			tarballDir, err := ioutil.TempDir("", "tarball")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tarballDir)

			tarballFile := filepath.Join(tarballDir, "test.tar")

			err = Tar(archiveSourceDir, tarballFile)
			Expect(err).ToNot(HaveOccurred())

			err = UntarProtect(tarballFile, targetDir, true, true,
				[]string{
					"/file-0",
					"/file-1",
					"/file-9999",
				}, tarModifierWrapperFunc)
			Expect(err).ToNot(HaveOccurred())

			Expect(fileHelper.Exists(filepath.Join(targetDir, "._cfg0001_file-0"))).Should(Equal(true))
		})
	})
})

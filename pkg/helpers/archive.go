// Copyright © 2019-2021 Ettore Di Giacinto <mudler@gentoo.org>
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

package helpers

import (
	"io"
	"os"

	"github.com/docker/docker/pkg/archive"
	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
)

func Tar(src, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	fs, err := archive.Tar(src, archive.Uncompressed)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = io.Copy(out, fs)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}
	return err
}

func UntarProtect(src, dst string, sameOwner, overwriteDirPerms bool, protectedFiles []string, modifier tarf.TarFileHandlerFunc) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	spec := tarf_specs.NewSpecFile()
	spec.SameOwner = sameOwner
	spec.EnableMutex = true
	spec.OverwritePerms = overwriteDirPerms
	spec.IgnoreFiles = []string{
		// prevent 'operation not permitted'
		"/dev",
	}

	tarformers := tarf.NewTarFormers(tarf.GetOptimusPrime().Config)
	tarformers.SetReader(in)

	if modifier != nil && len(protectedFiles) > 0 {
		tarformers.SetFileHandler(modifier)

		spec.TriggeredFiles = protectedFiles
	}

	return tarformers.RunTask(spec, dst)
}

// Untar just a wrapper around the docker functions
func Untar(src, dest string, sameOwner, overwriteDirPerms bool) error {
	return UntarProtect(src, dest, sameOwner, overwriteDirPerms, []string{}, nil)
}

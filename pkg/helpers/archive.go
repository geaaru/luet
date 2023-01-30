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

	. "github.com/geaaru/luet/pkg/config"

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

	spec := tarf_specs.NewSpecFile()
	spec.SameOwner = sameOwner
	spec.OverwritePerms = overwriteDirPerms
	spec.IgnoreRegexes = []string{
		// prevent 'operation not permitted'
		//"^/dev/",
	}
	spec.IgnoreFiles = []string{}
	spec.EnableMutex = LuetCfg.GetTarFlows().Mutex4Dirs
	spec.MaxOpenFiles = LuetCfg.GetTarFlows().MaxOpenFiles
	spec.BufferSize = LuetCfg.GetTarFlows().CopyBufferSize

	return UntarProtectSpec(
		src, dst, protectedFiles, modifier, spec,
	)
}

func prepareTarformers(in io.Reader, modifier tarf.TarFileHandlerFunc,
	spec *tarf_specs.SpecFile, protectedFiles []string) *tarf.TarFormers {
	tarformers := tarf.NewTarFormers(tarf.GetOptimusPrime().Config)
	tarformers.SetReader(in)

	if modifier != nil && len(protectedFiles) > 0 {
		tarformers.SetFileHandler(modifier)

		spec.TriggeredFiles = protectedFiles
	}

	return tarformers
}

func UntarProtectSpecCompress(dst string, protectedFiles []string,
	modifier tarf.TarFileHandlerFunc, spec *tarf_specs.SpecFile,
	compressStream io.Reader) error {

	tarformers := prepareTarformers(compressStream,
		modifier, spec, protectedFiles)

	return tarformers.RunTask(spec, dst)
}

func UntarProtectSpec(src, dst string, protectedFiles []string, modifier tarf.TarFileHandlerFunc, spec *tarf_specs.SpecFile) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tarformers := prepareTarformers(in, modifier, spec, protectedFiles)

	return tarformers.RunTask(spec, dst)
}

// Untar just a wrapper around the docker functions
func Untar(src, dest string, sameOwner, overwriteDirPerms bool) error {
	return UntarProtect(src, dest, sameOwner, overwriteDirPerms, []string{}, nil)
}

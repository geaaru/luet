// Copyright © 2021 Daniele Rondina <geaaru@sabayonlinux.org>
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

package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	fileHelper "github.com/geaaru/luet/pkg/helpers/file"

	"github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	capi "github.com/mudler/docker-companion/api"

	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
	"github.com/pkg/errors"
)

type Dockerv3 struct {
	*SimpleDocker
}

func NewDockerv3Backend() *Dockerv3 {
	return &Dockerv3{
		SimpleDocker: NewSimpleDockerBackend(),
	}
}

func (d *Dockerv3) deleteContainer(name string) {
	deleteargs := []string{"rm", name}

	Debug(":whale: deleting container with name" + name)
	out, err := exec.Command("docker", deleteargs...).CombinedOutput()
	if err != nil {
		Warning("Failed delete container " + name + " for image: " + string(out))
	} else {
		Debug("Container " + name + " removed.")
	}

}

func (s *Dockerv3) ImageDefinitionToTar(opts Options) error {
	if err := s.BuildImage(opts); err != nil {
		return errors.Wrap(err, "Failed building image")
	}
	if err := s.ExportImage(opts); err != nil {
		return errors.Wrap(err, "Failed exporting image")
	}
	if err := s.RemoveImage(opts); err != nil {
		return errors.Wrap(err, "Failed removing image")
	}
	return nil
}

func (d *Dockerv3) createTarFormers() *tarf.TarFormers {
	mutex.Lock()
	defer mutex.Unlock()

	// Create config
	cfg := tarf_specs.NewConfig(config.LuetCfg.Viper)
	cfg.GetGeneral().Debug = config.LuetCfg.GetGeneral().Debug
	cfg.GetLogging().Level = config.LuetCfg.GetLogging().Level

	ans := tarf.NewTarFormers(cfg)

	return ans
}

func (d *Dockerv3) ExportImage(opts Options) error {

	if opts.PackageDir == "" {
		return d.SimpleDocker.ExportImage(opts)
	}

	name := opts.ImageName
	dir := opts.Destination
	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}

	// Create the container from specified image
	createargs := []string{
		"create", name,
		"-c", "sleep", "1",
	}
	Debug(":whale: Creating container with name" + name)

	// Creating a fake container to use for the export.
	out, err := exec.Command("docker", createargs...).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed creating container for image: "+name)
	}
	idcontainer := strings.TrimRight(string(out), "\n")
	Debug(":whale: Container for image " + name + " (id " + idcontainer + ") created.")
	defer d.deleteContainer(idcontainer)

	// Prepare cp command where get stdout pipe.
	// The source path is in the format <container-id>:/path
	sourcePath := idcontainer + ":" + opts.PackageDir
	// The destpath must consider that dockers on cp get only
	// the final directory. So I need manually add the directory
	// if the package dir has multiple levels.
	paths := strings.Split(opts.PackageDir, "/")
	var destPath string
	if len(paths) > 2 {
		destPath = strings.Join(paths[:len(paths)-1], "/")
	} else {
		destPath = opts.PackageDir
	}
	// Append the destPath to the target dir
	destPath = dir + destPath[1:]

	Debug(fmt.Sprintf(":whale: Copy container file from %s to %s...",
		sourcePath, destPath))
	cpargs := []string{"cp", "-a", sourcePath, destPath}
	Info("Run docker " + strings.Join(cpargs, " "))
	exportCmd := exec.Command("docker", cpargs...)

	err = exportCmd.Start()
	if err != nil {
		return errors.Wrap(err, "Error on start docker cp command")
	}

	err = exportCmd.Wait()
	if err != nil {
		return errors.Wrap(err, "Failed wait command for image "+name)
	}

	if exportCmd.ProcessState.ExitCode() != 0 {
		return errors.New("Container export failed for image " + name)
	}

	Debug(":whale: Exported image:", name)

	return nil
}

func (b *Dockerv3) ExtractRootfs(opts Options, keepPerms bool) error {
	name := opts.ImageName
	dst := opts.Destination

	if !b.ImageExists(name) {
		if err := b.DownloadImage(opts); err != nil {
			return errors.Wrap(err, "failed pulling image "+name+" during extraction")
		}
	}

	tempexport, err := ioutil.TempDir(dst, "tmprootfs")
	if err != nil {
		return errors.Wrap(err, "Error met while creating tempdir for rootfs")
	}
	defer os.RemoveAll(tempexport) // clean up

	imageExport := filepath.Join(tempexport, "image.tar")
	if opts.PackageDir != "" {
		imageExport = dst
	}

	Spinner(22)
	defer SpinnerStop()

	if err := b.ExportImage(Options{
		ImageName:   name,
		Destination: imageExport,
		PackageDir:  opts.PackageDir,
	}); err != nil {
		return errors.Wrap(err, "failed while extracting rootfs for "+name)
	}

	SpinnerStop()

	if opts.PackageDir == "" {
		src := imageExport

		if src == "" && opts.ImageName != "" {
			tempUnpack, err := ioutil.TempDir(dst, "tempUnpack")
			if err != nil {
				return errors.Wrap(err, "Error met while creating tempdir for rootfs")
			}
			defer os.RemoveAll(tempUnpack) // clean up
			imageExport := filepath.Join(tempUnpack, "image.tar")
			if err := b.ExportImage(Options{ImageName: opts.ImageName, Destination: imageExport}); err != nil {
				return errors.Wrap(err, "while exporting image before extraction")
			}
			src = imageExport
		}

		rootfs, err := ioutil.TempDir(dst, "tmprootfs")
		if err != nil {
			return errors.Wrap(err, "Error met while creating tempdir for rootfs")
		}
		defer os.RemoveAll(rootfs) // clean up

		err = helpers.Untar(src, rootfs, keepPerms, true)
		if err != nil {
			return errors.Wrap(err, "Error met while unpacking rootfs")
		}

		manifest, err := fileHelper.Read(filepath.Join(rootfs, "manifest.json"))
		if err != nil {
			return errors.Wrap(err, "Error met while reading image manifest")
		}

		// Unpack all layers
		var manifestData []ManifestEntry

		if err := json.Unmarshal([]byte(manifest), &manifestData); err != nil {
			return errors.Wrap(err, "Error met while unmarshalling manifest")
		}

		layers_sha := []string{}

		for _, data := range manifestData {

			for _, l := range data.Layers {
				if strings.Contains(l, "layer.tar") {
					layers_sha = append(layers_sha, strings.Replace(l, "/layer.tar", "", -1))
				}
			}
		}
		// TODO: Drop capi in favor of the img approach already used in pkg/installer/repository
		export, err := capi.CreateExport(rootfs)
		if err != nil {
			return err
		}

		err = export.UnPackLayers(layers_sha, dst, "containerd")
		if err != nil {
			return err
		}
	}

	return nil
}

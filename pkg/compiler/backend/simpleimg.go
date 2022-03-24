// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
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
	"os"
	"os/exec"
	"strings"

	bus "github.com/geaaru/luet/pkg/bus"

	. "github.com/geaaru/luet/pkg/logger"

	"github.com/pkg/errors"
)

type SimpleImg struct{}

func NewSimpleImgBackend() *SimpleImg {
	return &SimpleImg{}
}

// TODO: Missing still: labels, and build args expansion
func (*SimpleImg) BuildImage(opts Options) error {
	name := opts.ImageName
	bus.Manager.Publish(bus.EventImagePreBuild, opts)

	buildarg := genBuildCommand(opts)

	Info(":tea: Building image " + name)

	cmd := exec.Command("img", buildarg...)
	cmd.Dir = opts.SourcePath
	err := runCommand(cmd)
	if err != nil {
		return err
	}
	bus.Manager.Publish(bus.EventImagePostBuild, opts)

	Info(":tea: Building image " + name + " done")

	return nil
}

func (*SimpleImg) RemoveImage(opts Options) error {
	name := opts.ImageName
	buildarg := []string{"rm", name}
	Spinner(22)
	defer SpinnerStop()
	out, err := exec.Command("img", buildarg...).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed removing image: "+string(out))
	}

	Info(":tea: Image " + name + " removed")
	return nil
}

func (*SimpleImg) DownloadImage(opts Options) error {
	name := opts.ImageName
	bus.Manager.Publish(bus.EventImagePrePull, opts)

	buildarg := []string{"pull", name}
	Debug(":tea: Downloading image " + name)

	Spinner(22)
	defer SpinnerStop()

	cmd := exec.Command("img", buildarg...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed downloading image: "+string(out))
	}

	Info(":tea: Image " + name + " downloaded")
	bus.Manager.Publish(bus.EventImagePostPull, opts)

	return nil
}
func (*SimpleImg) CopyImage(src, dst string) error {
	Spinner(22)
	defer SpinnerStop()

	Debug(":tea: Tagging image", src, dst)
	cmd := exec.Command("img", "tag", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed tagging image: "+string(out))
	}
	Info(":tea: Image " + dst + " tagged")

	return nil
}

func (*SimpleImg) ImageAvailable(imagename string) bool {
	return imageAvailable(imagename)
}

// ImageExists check if the given image is available locally
func (*SimpleImg) ImageExists(imagename string) bool {
	cmd := exec.Command("img", "ls")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	if strings.Contains(string(out), imagename) {
		return true
	}
	return false
}

func (s *SimpleImg) ImageDefinitionToTar(opts Options) error {
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

func (*SimpleImg) ExportImage(opts Options) error {
	name := opts.ImageName
	path := opts.Destination
	buildarg := []string{"save", "-o", path, name}
	Debug(":tea: Saving image " + name)

	Spinner(22)
	defer SpinnerStop()

	out, err := exec.Command("img", buildarg...).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed exporting image: "+string(out))
	}
	Info(":tea: Image " + name + " saved")
	return nil
}

// ExtractRootfs extracts the docker image content inside the destination
func (s *SimpleImg) ExtractRootfs(opts Options, keepPerms bool) error {
	name := opts.ImageName
	path := opts.Destination

	if !s.ImageExists(name) {
		if err := s.DownloadImage(opts); err != nil {
			return errors.Wrap(err, "failed pulling image "+name+" during extraction")
		}
	}

	os.RemoveAll(path)

	buildarg := []string{"unpack", "-o", path, name}
	Debug(":tea: Extracting image " + name)

	Spinner(22)
	defer SpinnerStop()

	out, err := exec.Command("img", buildarg...).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed extracting image: "+string(out))
	}
	Debug(":tea: Image " + name + " extracted")
	return nil
}

func (*SimpleImg) Push(opts Options) error {
	name := opts.ImageName
	bus.Manager.Publish(bus.EventImagePrePush, opts)

	pusharg := []string{"push", name}
	out, err := exec.Command("img", pusharg...).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed pushing image: "+string(out))
	}
	Info(":tea: Pushed image:", name)
	bus.Manager.Publish(bus.EventImagePostPush, opts)

	//Info(string(out))
	return nil
}

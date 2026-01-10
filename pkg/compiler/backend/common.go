/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package backend

import (
	"os/exec"

	"github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/pkg/errors"
)

const (
	ImgBackend      = "img"
	DockerBackend   = "docker"
	Dockerv2Backend = "dockerv2"
	Dockerv3Backend = "dockerv3"
)

func imageAvailable(image string) bool {
	_, err := crane.Digest(image)
	return err == nil
}

type Options struct {
	ImageName      string
	SourcePath     string
	DockerFileName string
	Destination    string
	Context        string
	BackendArgs    []string
	PackageDir     string
}

func runCommand(cmd *exec.Cmd) error {
	output := ""
	buffered := !config.LuetCfg.GetGeneral().ShowBuildOutput
	writer := NewBackendWriter(buffered)

	cmd.Stdout = writer
	cmd.Stderr = writer

	if buffered {
		Spinner(22)
		defer SpinnerStop()
	}

	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Failed starting command")
	}

	err = cmd.Wait()
	if err != nil {
		output = writer.GetCombinedOutput()
		return errors.Wrapf(err, "Failed running command: %s", output)
	}

	return nil
}

func genBuildCommand(opts Options) []string {
	context := opts.Context

	if context == "" {
		context = "."
	}
	buildarg := append(opts.BackendArgs, "-f", opts.DockerFileName, "-t", opts.ImageName, context)
	return append([]string{"build"}, buildarg...)
}

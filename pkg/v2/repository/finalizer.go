/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	box "github.com/macaroni-os/anise/pkg/box"
	. "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type AniseFinalizer struct {
	Shell     []string `json:"shell,omitempty" yaml:"shell,omitempty"`
	Install   []string `json:"install,omitempty" yaml:"install,omitempty"`
	Uninstall []string `json:"uninstall,omitempty" yaml:"uninstall,omitempty"`
}

func (f *AniseFinalizer) getShell() (string, []string) {
	var cmd string
	var args []string
	if len(f.Shell) == 0 {
		// Default to sh otherwise
		cmd = "sh"
		args = []string{"-c"}
	} else {
		cmd = f.Shell[0]
		if len(f.Shell) > 1 {
			args = f.Shell[1:]
		}
	}

	return cmd, args
}

func (f *AniseFinalizer) runCommand(cmd string, args, envs []string, script, targetRootfs string) error {
	toRun := append(args, script)
	Info(":shell: Executing finalizer on ", targetRootfs, cmd, toRun)
	if targetRootfs == string(os.PathSeparator) {
		cmd := exec.Command(cmd, toRun...)
		cmd.Env = envs
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrap(err, "Failed running command: "+string(stdoutStderr))
		}
		Info(string(stdoutStderr))
	} else {
		b := box.NewBox(cmd, toRun, []string{}, envs, targetRootfs,
			false, true, true, AniseCfg,
		)
		err := b.Run()
		if err != nil {
			return errors.Wrap(err, "Failed running command: ")
		}
	}

	return nil
}

func (f *AniseFinalizer) RunInstall(targetRootfs string) error {
	cmd, args := f.getShell()

	envs := AniseCfg.GetFinalizerEnvs()
	// Add ANISE_VERSION env so finalizer are able to know
	// what is the anise version and that the script is running
	// inside the anise command.
	envs = append(envs, fmt.Sprintf("ANISE_VERSION=%s", AniseVersion))

	// Add environment variable with the list of the subsets enabled
	envs = append(envs,
		fmt.Sprintf("ANISE_SUBSETS=%s",
			strings.Join(AniseCfg.Subsets.Enabled, " ")))

	for _, c := range f.Install {
		err := f.runCommand(cmd, args, envs, c, targetRootfs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *AniseFinalizer) RunUninstall(targetRootfs string) error {
	cmd, args := f.getShell()

	envs := AniseCfg.GetFinalizerEnvs()
	// Add ANISE_VERSION env so finalizer are able to know
	// what is the anise version and that the script is running
	// inside the anise command.
	envs = append(envs, fmt.Sprintf("ANISE_VERSION=%s", AniseVersion))

	// Add environment variable with the list of the subsets enabled
	envs = append(envs,
		fmt.Sprintf("ANISE_SUBSETS=%s",
			strings.Join(AniseCfg.Subsets.Enabled, " ")))

	for _, c := range f.Uninstall {
		err := f.runCommand(cmd, args, envs, c, targetRootfs)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewAniseFinalizerFromYaml(data []byte) (*AniseFinalizer, error) {
	var p AniseFinalizer
	err := yaml.Unmarshal(data, &p)
	if err != nil {
		return &p, err
	}
	return &p, err
}

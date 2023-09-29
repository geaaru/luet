/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	box "github.com/geaaru/luet/pkg/box"
	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"
)

type LuetFinalizer struct {
	Shell     []string `json:"shell,omitempty" yaml:"shell,omitempty"`
	Install   []string `json:"install,omitempty" yaml:"install,omitempty"`
	Uninstall []string `json:"uninstall,omitempty" yaml:"uninstall,omitempty"`
}

func (f *LuetFinalizer) getShell() (string, []string) {
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

func (f *LuetFinalizer) runCommand(cmd string, args, envs []string, script, targetRootfs string) error {
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
		b := box.NewBox(cmd, toRun, []string{}, envs, targetRootfs, false, true, true)
		err := b.Run()
		if err != nil {
			return errors.Wrap(err, "Failed running command: ")
		}
	}

	return nil
}

func (f *LuetFinalizer) RunInstall(targetRootfs string) error {
	cmd, args := f.getShell()

	envs := LuetCfg.GetFinalizerEnvs()
	// Add LUET_VERSION env so finalizer are able to know
	// what is the luet version and that the script is running
	// inside the luet command.
	envs = append(envs, fmt.Sprintf("LUET_VERSION=%s", LuetVersion))

	// Add environment variable with the list of the subsets enabled
	envs = append(envs,
		fmt.Sprintf("ANISE_SUBSETS=%s",
			strings.Join(LuetCfg.Subsets.Enabled, " ")))

	for _, c := range f.Install {
		err := f.runCommand(cmd, args, envs, c, targetRootfs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *LuetFinalizer) RunUninstall(targetRootfs string) error {
	cmd, args := f.getShell()

	envs := LuetCfg.GetFinalizerEnvs()
	// Add LUET_VERSION env so finalizer are able to know
	// what is the luet version and that the script is running
	// inside the luet command.
	envs = append(envs, fmt.Sprintf("LUET_VERSION=%s", LuetVersion))

	// Add environment variable with the list of the subsets enabled
	envs = append(envs,
		fmt.Sprintf("ANISE_SUBSETS=%s",
			strings.Join(LuetCfg.Subsets.Enabled, " ")))

	for _, c := range f.Uninstall {
		err := f.runCommand(cmd, args, envs, c, targetRootfs)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewLuetFinalizerFromYaml(data []byte) (*LuetFinalizer, error) {
	var p LuetFinalizer
	err := yaml.Unmarshal(data, &p)
	if err != nil {
		return &p, err
	}
	return &p, err
}

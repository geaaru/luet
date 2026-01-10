/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package installer

import (
	"fmt"
	"os"
	"os/exec"

	box "github.com/macaroni-os/anise/pkg/box"
	. "github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

type AniseFinalizer struct {
	Shell     []string `json:"shell"`
	Install   []string `json:"install"`
	Uninstall []string `json:"uninstall"` // TODO: Where to store?
}

func (f *AniseFinalizer) RunInstall(s *System) error {
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

	envs := AniseCfg.GetFinalizerEnvs()
	// Add LUET_VERSION env so finalizer are able to know
	// what is the luet version and that the script is running
	// inside the luet command.
	envs = append(envs, fmt.Sprintf("LUET_VERSION=%s", AniseVersion))

	for _, c := range f.Install {
		toRun := append(args, c)
		Info(":shell: Executing finalizer on ", s.Target, cmd, toRun)
		if s.Target == string(os.PathSeparator) {
			cmd := exec.Command(cmd, toRun...)
			cmd.Env = envs
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				return errors.Wrap(err, "Failed running command: "+string(stdoutStderr))
			}
			Info(string(stdoutStderr))
		} else {
			b := box.NewBox(cmd, toRun, []string{}, envs, s.Target,
				false, true, true, AniseCfg)
			err := b.Run()
			if err != nil {
				return errors.Wrap(err, "Failed running command: ")
			}
		}
	}
	return nil
}

// TODO: We don't store uninstall finalizers ?!
func (f *AniseFinalizer) RunUnInstall() error {
	for _, c := range f.Uninstall {
		Debug("finalizer:", "sh", "-c", c)
		cmd := exec.Command("sh", "-c", c)
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrap(err, "Failed running command: "+string(stdoutStderr))
		}
		Info(string(stdoutStderr))
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

/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package box

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/macaroni-os/anise/pkg/config"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	. "github.com/macaroni-os/anise/pkg/logger"
)

type FchrootBox struct {
	*DefaultBox
	Config *config.AniseConfig
}

func NewFchrootBox(cmd string, args, hostmounts, env []string,
	rootfs string, stdin, stdout, stderr bool,
	cfg *config.AniseConfig) Box {
	return &FchrootBox{
		DefaultBox: &DefaultBox{
			Stdin:      stdin,
			Stdout:     stdout,
			Stderr:     stderr,
			Cmd:        cmd,
			Args:       args,
			Root:       rootfs,
			HostMounts: hostmounts,
			Env:        env,
		},
		Config: cfg,
	}
}

func (f *FchrootBox) Exec() error {

	binds := []string{}

	if len(f.HostMounts) > 0 {
		for _, hostMount := range f.HostMounts {
			target := hostMount
			if strings.Contains(hostMount, ":") {
				dest := strings.Split(hostMount, ":")
				if len(dest) != 2 {
					return fmt.Errorf(
						"Invalid arguments for mount, it can be: fullpath, or source:target")
				}
				hostMount = dest[0]
				target = dest[1]
			}

			binds = append(binds,
				fmt.Sprintf("%s:%s", hostMount, target))
		}
	}

	fchroot := fileHelper.TryResolveBinaryAbsPath("fchroot")
	fchrootEntrypoint := []string{fchroot}

	fopts := f.Config.GetBox().FchrootOpts
	if fopts == nil {
		fopts = &config.FchrootOpts{
			Verbose: false,
			Debug:   false,
			Cpu:     "",
			NoBind:  false,
		}
	}

	// Add fchroot flags
	fchrootEntrypoint = append(fchrootEntrypoint, fopts.GetFlags(binds)...)

	// Add rootfs path
	fchrootEntrypoint = append(fchrootEntrypoint, f.Root)

	cmds := []string{}
	cmds = append(cmds, fchrootEntrypoint...)
	cmds = append(cmds, f.Cmd)
	if len(f.Args) > 0 {
		cmds = append(cmds, f.Args...)
	}

	chrootCommand := exec.Command(cmds[0], cmds[1:]...)

	if f.Stdin {
		chrootCommand.Stdin = os.Stdin
	}
	if f.Stderr {
		chrootCommand.Stderr = os.Stderr
	}
	if f.Stdout {
		chrootCommand.Stdout = os.Stdout
	}

	chrootCommand.Env = append(f.Env,
		"ANISE_CHROOT=1")

	err := chrootCommand.Start()
	if err != nil {
		Error("Error on start command: " + err.Error())
		return err
	}

	err = chrootCommand.Wait()
	if err != nil {
		Error("Error on waiting command: " + err.Error())
		return err
	}

	res := chrootCommand.ProcessState.ExitCode()

	Debug(fmt.Sprintf(
		":high-speed_train: Exiting [%d]", res))

	if res != 0 {
		return fmt.Errorf("command exiting with", res)
	}

	return nil
}

func (f *FchrootBox) Run() error {

	if !fileHelper.Exists(f.Root) {
		return fmt.Errorf(f.Root + " does not exist")
	}

	// Fchroot requires the path /proc, /sys and /dev
	// I create the directories if they are missed.
	for _, p := range []string{"/proc", "/sys", "/dev"} {
		dir := filepath.Join(f.Root, p)
		if !fileHelper.Exists(dir) {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	return f.Exec()
}

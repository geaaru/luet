/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package helpers

import (
	"os"
	"os/exec"
	"os/user"
	"syscall"

	"github.com/pkg/errors"
)

// This allows a multi-platform switch in the future
func Exec(cmd string, args []string, env []string) error {
	path, err := exec.LookPath(cmd)
	if err != nil {
		return errors.Wrap(err, "Could not find binary in path: "+cmd)
	}
	return syscall.Exec(path, args, env)
}

func GetHomeDir() (ans string) {
	// os/user doesn't work in from scratch environments
	u, err := user.Current()
	if err == nil {
		ans = u.HomeDir
	} else {
		ans = ""
	}
	if os.Getenv("HOME") != "" {
		ans = os.Getenv("HOME")
	}
	return ans
}

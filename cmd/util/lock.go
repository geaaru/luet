/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package util

import (
	"os"
	"path/filepath"

	config "github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	fhelpers "github.com/geaaru/luet/pkg/helpers/file"

	"github.com/gofrs/flock"
)

var LockedCommands = []string{"install", "uninstall", "upgrade"}

type LockGuard struct {
	Lockfile *flock.Flock
}

func NewLockGuard() *LockGuard {
	return &LockGuard{
		Lockfile: nil,
	}
}

func (l *LockGuard) TryLock(cmd string, cfg *config.LuetConfig) (bool, error) {
	envNoLock := os.Getenv("LUET_NOLOCK") == "true"
	if !envNoLock && helpers.ContainsElem(&LockedCommands, cmd) {
		// Using the rootfs directory for locking.
		// This permits to avoid locking between different rootfs.
		fpath := cfg.GetLockFilePath("luet.lock")
		l.Lockfile = flock.New(fpath)

		if !fhelpers.Exists(filepath.Dir(fpath)) {
			err := os.MkdirAll(filepath.Dir(fpath), 0755)
			if err != nil {
				return false, err
			}
		}

		return l.Lockfile.TryLock()
	} else {
		return true, nil
	}
}

func (l *LockGuard) Locked() (ans bool) {
	ans = false
	if l.Lockfile != nil {
		ans = l.Lockfile.Locked()
	}
	return
}

func (l *LockGuard) Unlock() error {
	if l.Lockfile != nil {
		return l.Lockfile.Unlock()
	}
	return nil
}

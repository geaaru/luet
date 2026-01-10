/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package util

import (
	"os"
	"path/filepath"

	config "github.com/macaroni-os/anise/pkg/config"
	"github.com/macaroni-os/anise/pkg/helpers"
	fhelpers "github.com/macaroni-os/anise/pkg/helpers/file"

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

func (l *LockGuard) TryLock(cmd string, cfg *config.AniseConfig) (bool, error) {
	envNoLock := os.Getenv("ANISE_NOLOCK") == "true"
	if !envNoLock && helpers.ContainsElem(&LockedCommands, cmd) {
		// Using the rootfs directory for locking.
		// This permits to avoid locking between different rootfs.
		fpath := cfg.GetLockFilePath("anise.lock")
		l.Lockfile = flock.New(fpath)

		lockDir := filepath.Dir(fpath)
		if !fhelpers.Exists(lockDir) {
			// Check if lock is a link to directory that doesn't exist.
			if !fhelpers.ExistsLink(lockDir) {
				err := os.MkdirAll(filepath.Dir(fpath), 0755)
				if err != nil {
					return false, err
				}
			} else {
				// POST: The lock directory is a link. Checking if the link exists.
				linkedLockDir, err := os.Readlink(lockDir)
				if err != nil {
					return false, err
				}

				// If the link is abs path
				if !filepath.IsAbs(linkedLockDir) {
					linkedLockDir = filepath.Join(filepath.Dir(lockDir), linkedLockDir)
				}

				if !fhelpers.Exists(linkedLockDir) {
					err := os.MkdirAll(linkedLockDir, 0755)
					if err != nil {
						return false, err
					}
				}

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

func (l *LockGuard) Unlock(cfg *config.AniseConfig) error {
	if l.Lockfile != nil {
		if l.Locked() {
			// POST: Try to remove the file only when is been locked.
			fpath := cfg.GetLockFilePath("anise.lock")
			defer os.RemoveAll(fpath)
		}
		return l.Lockfile.Unlock()
	}
	return nil
}

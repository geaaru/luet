/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	. "github.com/macaroni-os/anise/pkg/logger"
)

func (a *ArtifactsManager) CleanLocalPackagesCache() error {
	var cleaned int = 0

	// Check if cache dir exists
	if fileHelper.Exists(a.Config.GetSystem().GetSystemPkgsCacheDirPath()) {

		files, err := ioutil.ReadDir(a.Config.GetSystem().GetSystemPkgsCacheDirPath())
		if err != nil {
			return fmt.Errorf("Error on read cachedir: %s", err.Error())
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if a.Config.GetGeneral().Debug {
				Info("Removing ", file.Name())
			}

			err := os.RemoveAll(
				filepath.Join(a.Config.GetSystem().GetSystemPkgsCacheDirPath(), file.Name()))
			if err != nil {
				return fmt.Errorf("Error on removing %s", file.Name())
			}
			cleaned++
		}
	}

	Info("Cleaned: ", cleaned, "packages.")

	return nil
}

func (a *ArtifactsManager) PurgeLocalReposCache() error {
	reposDir := a.Config.GetSystem().GetSystemReposDirPath()
	cnt := 0

	Debug("Repositories dir:", reposDir)

	if fileHelper.Exists(reposDir) {

		files, err := ioutil.ReadDir(reposDir)
		if err != nil {
			return fmt.Errorf("Error on read reposdir: %s", err.Error())
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			d := filepath.Join(reposDir, file.Name())

			err := os.RemoveAll(d)
			if err != nil {
				return fmt.Errorf("Error on removing dir %s", d)
			}

			cnt++
		}

		Info("Repos Cleaned: ", cnt)

	}

	return nil
}

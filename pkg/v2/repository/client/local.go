/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package client

import (
	"os"
	"path"
	"path/filepath"

	"github.com/macaroni-os/anise/pkg/config"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	. "github.com/macaroni-os/anise/pkg/logger"
	"github.com/macaroni-os/anise/pkg/v2/compiler/types/artifact"
)

type LocalClient struct {
	Repository *config.AniseRepository
}

func NewLocalClient(r *config.AniseRepository) *LocalClient {
	return &LocalClient{Repository: r}
}

func (c *LocalClient) DownloadArtifact(a *artifact.PackageArtifact, msg string) error {
	var err error

	rootfs := ""
	artifactName := path.Base(a.Path)
	cacheFile := filepath.Join(config.AniseCfg.GetSystem().GetSystemPkgsCacheDirPath(), artifactName)

	if !config.AniseCfg.ConfigFromHost {
		rootfs, err = config.AniseCfg.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	// Check if file is already in cache
	if fileHelper.Exists(cacheFile) {
		Debug("Use artifact", artifactName, "from cache.")
	} else {
		ok := false
		for _, uri := range c.Repository.Urls {

			uri = filepath.Join(rootfs, uri)

			Debug("Downloading artifact", artifactName, "from", uri)

			//defer os.Remove(file.Name())
			err = fileHelper.CopyFile(filepath.Join(uri, artifactName), cacheFile)
			if err != nil {
				continue
			}
			ok = true
			break
		}

		if !ok {
			return err
		}
	}

	a.CachePath = cacheFile
	return nil
}

func (c *LocalClient) DownloadFile(name string) (string, error) {
	var err error
	var file *os.File = nil

	rootfs := ""

	if !config.AniseCfg.ConfigFromHost {
		rootfs, err = config.AniseCfg.GetSystem().GetRootFsAbs()
		if err != nil {
			return "", err
		}
	}

	ok := false
	for _, uri := range c.Repository.Urls {

		uri = filepath.Join(rootfs, uri)

		Debug("Downloading file", name, "from", uri)
		file, err = config.AniseCfg.GetSystem().TempFile("localclient")
		if err != nil {
			continue
		}
		//defer os.Remove(file.Name())

		err = fileHelper.CopyFile(filepath.Join(uri, name), file.Name())
		if err != nil {
			continue
		}
		ok = true
		break
	}

	if ok {
		return file.Name(), nil
	}

	return "", err
}

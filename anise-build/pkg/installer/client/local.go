/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package client

import (
	"os"
	"path"
	"path/filepath"

	"github.com/macaroni-os/anise/pkg/compiler/types/artifact"
	"github.com/macaroni-os/anise/pkg/config"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	. "github.com/macaroni-os/anise/pkg/logger"
)

type LocalClient struct {
	RepoData RepoData
}

func NewLocalClient(r RepoData) *LocalClient {
	return &LocalClient{RepoData: r}
}

func (c *LocalClient) DownloadArtifact(a *artifact.PackageArtifact) (*artifact.PackageArtifact, error) {
	var err error

	rootfs := ""
	artifactName := path.Base(a.Path)
	cacheFile := filepath.Join(config.AniseCfg.GetSystem().GetSystemPkgsCacheDirPath(), artifactName)

	if !config.AniseCfg.ConfigFromHost {
		rootfs, err = config.AniseCfg.GetSystem().GetRootFsAbs()
		if err != nil {
			return nil, err
		}
	}

	// Check if file is already in cache
	if fileHelper.Exists(cacheFile) {
		Debug("Use artifact", artifactName, "from cache.")
	} else {
		ok := false
		for _, uri := range c.RepoData.Urls {

			uri = filepath.Join(rootfs, uri)

			Info("Downloading artifact", artifactName, "from", uri)

			//defer os.Remove(file.Name())
			err = fileHelper.CopyFile(filepath.Join(uri, artifactName), cacheFile)
			if err != nil {
				continue
			}
			ok = true
			break
		}

		if !ok {
			return nil, err
		}
	}

	newart := a
	newart.Path = cacheFile
	return newart, nil
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
	for _, uri := range c.RepoData.Urls {

		uri = filepath.Join(rootfs, uri)

		Info("Downloading file", name, "from", uri)
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

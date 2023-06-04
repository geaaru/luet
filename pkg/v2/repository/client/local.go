/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package client

import (
	"os"
	"path"
	"path/filepath"

	"github.com/geaaru/luet/pkg/config"
	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
	. "github.com/geaaru/luet/pkg/logger"
	"github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
)

type LocalClient struct {
	Repository *config.LuetRepository
}

func NewLocalClient(r *config.LuetRepository) *LocalClient {
	return &LocalClient{Repository: r}
}

func (c *LocalClient) DownloadArtifact(a *artifact.PackageArtifact, msg string) error {
	var err error

	rootfs := ""
	artifactName := path.Base(a.Path)
	cacheFile := filepath.Join(config.LuetCfg.GetSystem().GetSystemPkgsCacheDirPath(), artifactName)

	if !config.LuetCfg.ConfigFromHost {
		rootfs, err = config.LuetCfg.GetSystem().GetRootFsAbs()
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

	if !config.LuetCfg.ConfigFromHost {
		rootfs, err = config.LuetCfg.GetSystem().GetRootFsAbs()
		if err != nil {
			return "", err
		}
	}

	ok := false
	for _, uri := range c.Repository.Urls {

		uri = filepath.Join(rootfs, uri)

		Debug("Downloading file", name, "from", uri)
		file, err = config.LuetCfg.GetSystem().TempFile("localclient")
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

/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package repository

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	"gopkg.in/yaml.v3"
)

func LoadRepositories(c *LuetConfig) error {
	var regexRepo = regexp.MustCompile(`.yml$|.yaml$`)
	var err error
	rootfs := ""

	// Respect the rootfs param on read repositories
	if !c.ConfigFromHost {
		rootfs, err = c.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	for _, rdir := range c.RepositoriesConfDir {

		rdir = filepath.Join(rootfs, rdir)

		Debug("Parsing Repository Directory", rdir, "...")

		dirEntries, err := os.ReadDir(rdir)
		if err != nil {
			Debug("Skip dir", rdir, ":", err.Error())
			continue
		}

		for _, file := range dirEntries {
			if file.IsDir() {
				continue
			}

			if !regexRepo.MatchString(file.Name()) {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			if strings.HasPrefix(file.Name(), "._cfg") {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			repoFile := path.Join(rdir, file.Name())

			content, err := os.ReadFile(repoFile)
			if err != nil {
				Warning("On read file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			r, err := LoadRepository(content)
			if err != nil {
				Warning("On parse file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if r.Name == "" || len(r.Urls) == 0 || r.Type == "" {
				Warning("Invalid repository ", file.Name())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if !r.Cached {
				Warning("In memory repositories will be dropped and no more supported.")
				Warning("The repository " + r.Name + " is forced to caching.")
				r.Cached = true
			}

			r.File = repoFile

			c.AddSystemRepository(r)
		}
	}
	return nil
}

func LoadRepository(data []byte) (*LuetRepository, error) {
	ans := NewEmptyLuetRepository()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}
	return ans, nil
}

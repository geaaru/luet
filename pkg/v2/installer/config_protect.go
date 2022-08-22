/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	. "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
)

func LoadConfigProtectConfs(c *LuetConfig) error {
	var regexConfs = regexp.MustCompile(`.yml$`)
	var err error

	rootfs := ""

	// Respect the rootfs param on read repositories
	if !c.ConfigFromHost {
		rootfs, err = c.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	for _, cdir := range c.ConfigProtectConfDir {
		cdir = filepath.Join(rootfs, cdir)

		Debug("Parsing Config Protect Directory", cdir, "...")

		files, err := ioutil.ReadDir(cdir)
		if err != nil {
			Debug("Skip dir", cdir, ":", err.Error())
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !regexConfs.MatchString(file.Name()) {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			content, err := ioutil.ReadFile(path.Join(cdir, file.Name()))
			if err != nil {
				Warning("On read file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			r, err := LoadConfigProtectConFile(file.Name(), content)
			if err != nil {
				Warning("On parse file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if r.Name == "" || len(r.Directories) == 0 {
				Warning("Invalid config protect file", file.Name())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			c.AddConfigProtectConfFile(r)
		}
	}
	return nil

}

func LoadConfigProtectConFile(filename string, data []byte) (*ConfigProtectConfFile, error) {
	ans := NewConfigProtectConfFile(filename)
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}
	return ans, nil
}

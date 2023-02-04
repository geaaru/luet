/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package mask

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

func NewPackagesMaskManager(c *config.LuetConfig) *PackagesMaskManager {
	return &PackagesMaskManager{
		Config: c,
		Files:  []*PackageMaskFile{},
	}
}

func (m *PackagesMaskManager) LoadFiles() error {
	var regexRepo = regexp.MustCompile(`.yml$|.yaml$`)
	var err error
	rootfs := ""

	// Respect the rootfs param on read repositories
	if !m.Config.ConfigFromHost {
		rootfs, err = m.Config.GetSystem().GetRootFsAbs()
		if err != nil {
			return err
		}
	}

	if len(m.Config.PackagesMaskDir) > 0 {
		for _, mdir := range m.Config.PackagesMaskDir {
			mdir = filepath.Join(rootfs, mdir)

			Debug("Parsing Packages Mask Directory", mdir, "...")

			files, err := ioutil.ReadDir(mdir)
			if err != nil {
				Debug("Skip dir", mdir, ":", err.Error())
				continue
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				if !regexRepo.MatchString(file.Name()) {
					Debug("File", file.Name(), "skipped.")
					continue
				}

				content, err := os.ReadFile(path.Join(mdir, file.Name()))
				if err != nil {
					Warning("On read file", file.Name(), ":", err.Error())
					Warning("File", file.Name(), "skipped.")
					continue
				}

				pmf, err := NewPackageMaskFileFromData(file.Name(), content)
				if err != nil {
					Warning("On parser file", file.Name(), ":", err.Error())
					Warning("File", file.Name(), "skipped.")
					continue
				}
				content = nil

				if !pmf.Enabled {
					Debug(fmt.Sprintf(
						"packages mask file %s is disable. Skipped.",
						file.Name(),
					))
					continue
				}

				err = pmf.BuildMap()
				if err != nil {
					Error(fmt.Sprintf("Mask file %s broken: %s.",
						file.Name(), err.Error()))
					return err
				}

				m.Files = append(m.Files, pmf)
			}

		}
	}

	return nil
}

func (m *PackagesMaskManager) IsMasked(repo string, p *gentoo.GentooPackage) (bool, error) {
	if len(m.Files) > 0 {
		for idx, _ := range m.Files {
			masked, err := m.Files[idx].Mask(repo, p)
			if err != nil || masked {
				return true, err
			}
		}
	}

	return false, nil
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	cfg "github.com/geaaru/luet/pkg/config"
	"github.com/geaaru/luet/pkg/helpers"
	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	"github.com/geaaru/luet/pkg/tree"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	repos "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/pkg/errors"
)

type ArtifactsManager struct {
	Config   *cfg.LuetConfig
	Database pkg.PackageDatabase

	sync.Mutex

	// Temporary struct to review
	fileIndex map[string]*pkg.DefaultPackage
}

func NewArtifactsManager(config *cfg.LuetConfig) *ArtifactsManager {
	return &ArtifactsManager{
		Config:    config,
		Database:  nil,
		fileIndex: nil,
	}
}

func (m *ArtifactsManager) setup() {
	if m.Database == nil {
		m.Database = m.Config.GetSystemDB()
	}
}

func (m *ArtifactsManager) Close() {
	if m.Database != nil {
		m.Database.Close()
	}
}

func (m *ArtifactsManager) DownloadPackage(p *artifact.PackageArtifact, r *repos.WagonRepository) error {
	c := r.Client()
	if c == nil {
		return errors.New("No client could be generated from repository")
	}

	err := c.DownloadArtifact(p)
	if err != nil {
		return errors.Wrap(err, "Error on download artifact")
	}

	err = p.Verify()
	if err != nil {
		return errors.Wrap(err,
			"Artifact integrity check failure for file "+p.CachePath)
	}

	return nil
}

func (m *ArtifactsManager) removePackageFiles(s *repos.Stone, targetRootfs string, preserveSystemEssentialData bool) error {
	var cp *cfg.ConfigProtect
	var err error
	annotationDir := ""

	p := s.ToPackage()

	if !m.Config.ConfigProtectSkip {
		if p.HasAnnotation(string(pkg.ConfigProtectAnnnotation)) {
			dir, ok := p.GetAnnotations()[string(pkg.ConfigProtectAnnnotation)].(string)
			if ok {
				annotationDir = dir
			}
		}

		cp = cfg.NewConfigProtect(annotationDir)
		cp.Map(s.Files)
	}

	toRemove, dirs2Remove, notPresent := fileHelper.OrderFiles(targetRootfs, s.Files)

	mapDirs := make(map[string]int, 0)
	for _, d := range dirs2Remove {
		mapDirs[d] = 1
	}

	// Remove from target
	for _, f := range toRemove {
		target := filepath.Join(targetRootfs, f)

		if !m.Config.ConfigProtectSkip && cp.Protected(f) {
			Debug("Preserving protected file:", f)
			continue
		}

		Debug("Removing", target)
		if preserveSystemEssentialData &&
			strings.HasPrefix(f, m.Config.GetSystem().GetSystemPkgsCacheDirPath()) ||
			strings.HasPrefix(f, m.Config.GetSystem().GetSystemRepoDatabaseDirPath()) {
			Warning("Preserve ", f,
				" which is required by luet ( you have to delete it manually if you really need to)")
			continue
		}

		fi, err := os.Lstat(target)
		if err != nil {
			Warning("File not found (it was before?)", err.Error())
			continue
		}
		switch mode := fi.Mode(); {
		case mode.IsDir():
			files, err := ioutil.ReadDir(target)
			if err != nil {
				Warning("Failed reading folder", target, err.Error())
			}
			if len(files) != 0 {
				Info("DROPPED = Preserving not-empty folder", target)
				continue
			}
		}

		if err = os.Remove(target); err != nil {
			Warning("Failed removing file (maybe not present in the system target anymore ?)", target, err.Error())
		}

		// Add subpaths of the file to ensure that all dirs
		// are injected for the prune phase. (NOTE: i'm not sure about this)
		dirname := filepath.Dir(target)
		words := strings.Split(dirname, string(os.PathSeparator))

		for i := len(words); i > 1; i-- {
			cpath := strings.Join(words[0:i], string(os.PathSeparator))
			if _, ok := mapDirs[cpath]; !ok {
				mapDirs[cpath] = 1
			}
		}
	}

	for _, f := range notPresent {
		target := filepath.Join(targetRootfs, f)

		if !m.Config.ConfigProtectSkip && cp.Protected(f) {
			Debug("Preserving protected file:", f)
			continue
		}

		if err = os.Remove(target); err != nil {
			Debug("Failed removing file (not present in the system target)", target, err.Error())
		}
	}

	// Sorting the dirs from the mapDirs keys
	dirs2Remove = []string{}
	for k, _ := range mapDirs {
		dirs2Remove = append(dirs2Remove, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs2Remove)))

	Debug("Directories tagged for the check and remove", len(dirs2Remove))

	// Check if directories could be removed.
	for _, f := range dirs2Remove {
		target := filepath.Join(targetRootfs, f)

		if preserveSystemEssentialData &&
			strings.HasPrefix(f, m.Config.GetSystem().GetSystemPkgsCacheDirPath()) ||
			strings.HasPrefix(f, m.Config.GetSystem().GetSystemRepoDatabaseDirPath()) {
			Warning("Preserve ", f,
				" which is required by luet ( you have to delete it manually if you really need to)")
			continue
		}

		if !m.Config.ConfigProtectSkip && cp.Protected(f) {
			Debug("Preserving protected file:", f)
			continue
		}

		files, err := ioutil.ReadDir(target)
		if err != nil {
			Warning("Failed reading folder", target, err.Error())
		}
		Debug("Removing dir", target, "if empty: files ", len(files), ".")

		if len(files) != 0 {
			Debug("Preserving not-empty folder", target)
			continue
		}

		if err = os.Remove(target); err != nil {
			Debug("Failed removing file (not present in the system target)", target, err.Error())
		}
	}

	return nil
}

func (m *ArtifactsManager) RemovePackage(s *repos.Stone, targetRootfs string, preserveSystemEssentialData bool) error {
	m.setup()

	err := m.removePackageFiles(s, targetRootfs, preserveSystemEssentialData)
	if err != nil {
		return err
	}

	p := s.ToPackage()

	err = m.Database.RemovePackageFiles(p)
	if err != nil {
		return errors.Wrap(err, "Failed removing package files from database")
	}
	err = m.Database.RemovePackage(p)
	if err != nil {
		return errors.Wrap(err, "Failed removing package from database")
	}

	Info(":recycle: ", fmt.Sprintf("%20s", p.GetFingerPrint()), "Removed :heavy_check_mark:")
	return nil
}

func (m *ArtifactsManager) ReinstallPackage(
	s *repos.Stone,
	p *artifact.PackageArtifact,
	r *repos.WagonRepository,
	targetRootfs string,
	preserveSystemEssentialData bool) error {

	if p.Runtime == nil {
		return errors.New("Artifact without Runtime package definition")
	}

	m.setup()

	err := m.removePackageFiles(s, targetRootfs, preserveSystemEssentialData)
	if err != nil {
		return err
	}

	return m.InstallPackage(p, r, targetRootfs)
}

func (m *ArtifactsManager) InstallPackage(p *artifact.PackageArtifact, r *repos.WagonRepository, targetRootfs string) error {

	if p.Runtime == nil {
		return errors.New("Artifact without Runtime package definition")
	}

	start := time.Now()

	// PRE: the package is already downloaded and available in cache
	//      directory.

	// TODO: Check if it's needed an option to disable the
	//       second argument related to subsets feature.
	//       For now i enable it always.
	err := p.Unpack(targetRootfs, true)
	if err != nil {
		return errors.Wrap(err,
			fmt.Sprintf("Unpack package %s failed", p.Runtime.HumanReadableString()))
	}

	Debug(fmt.Sprintf("Unpack package %s completed in %d µs.",
		p.Runtime.HumanReadableString(),
		time.Now().Sub(start).Nanoseconds()/1e3))

	return nil
}

func (m *ArtifactsManager) RegisterPackage(p *artifact.PackageArtifact, r *repos.WagonRepository) error {

	if p.Runtime == nil {
		return errors.New("Artifact without Runtime package definition")
	}

	m.setup()

	start := time.Now()

	// Set package files on local database
	err := m.Database.SetPackageFiles(
		&pkg.PackageFile{
			PackageFingerprint: p.Runtime.GetFingerPrint(),
			Files:              p.Files,
		},
	)
	if err != nil {
		return errors.Wrap(err, "Register package files on database")
	}

	// NOTE: for now postpone the registration of the package
	//       on the database

	_, err = m.Database.CreatePackage(p.Runtime)
	if err != nil {
		return errors.Wrap(err, "Failed register package")
	}

	Debug(fmt.Sprintf("Register package %s completed in %d µs.",
		p.Runtime.HumanReadableString(),
		time.Now().Sub(start).Nanoseconds()/1e3))

	return nil
}

func (m *ArtifactsManager) CheckFileConflicts(
	toInstall *[]*artifact.PackageArtifact,
	checkSystem bool,
	safeCheck bool,
	targetRootfs string,
) error {

	m.setup()

	Info("Checking for file conflicts...")
	start := time.Now()

	filesToInstall := make(map[string]string, 0)

	// NOTE: Instead of load in memory the list
	//       of the files of every installed package
	//       and consume high memory I do it only for
	//       the list of the packages to install.
	//       The check validate if a package file is present
	//       on target system or between the list of
	//       files of the packages to install.
	for _, a := range *toInstall {
		for _, f := range a.Files {
			if pkg, ok := filesToInstall[f]; ok {
				if safeCheck {
					Warning(fmt.Errorf(
						"file %s conflict between package %s and %s",
						f, pkg, a.CompileSpec.Package.HumanReadableString(),
					))
				} else {
					return fmt.Errorf(
						"file %s conflict between package %s and %s",
						f, pkg, a.CompileSpec.Package.HumanReadableString(),
					)
				}
			}

			filesToInstall[f] = a.CompileSpec.Package.HumanReadableString()

			if checkSystem {
				tFile := filepath.Join(targetRootfs, f)

				// Check if the file is present on the target path.
				if fileHelper.Exists(tFile) {
					exists, p, err := m.ExistsPackageFile(f)
					if err != nil {
						return errors.Wrap(err, "failed checking into system db")
					}
					if exists {
						if safeCheck {

							Warning(fmt.Errorf(
								"file conflict between '%s' and '%s' ( file: %s )",
								p.HumanReadableString(),
								a.Runtime.HumanReadableString(),
								f,
							))
						} else {
							return fmt.Errorf(
								"file conflict between '%s' and '%s' ( file: %s )",
								p.HumanReadableString(),
								a.Runtime.HumanReadableString(),
								f,
							)
						}
					}
				}
			}

		} // end for a.Files

	} // end for toInstall

	Info(fmt.Sprintf("Check for file conflicts completed in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))

	return nil
}

func (m *ArtifactsManager) ExecuteFinalizer(
	a *artifact.PackageArtifact,
	r *repos.WagonRepository,
	postInstall bool,
	targetRootfs string) error {

	repoTreefs := r.GetTreePath(m.Config.GetSystem().GetSystemReposDirPath())
	pkgdir := a.GetPackageTreePath(repoTreefs)
	finalizeFile := filepath.Join(pkgdir, tree.FinalizerFile)
	defFile := filepath.Join(pkgdir, pkg.PackageDefinitionFile)

	if a.Runtime == nil && a.CompileSpec.Package == nil {
		return errors.New("Invalid artifact without Package metadata")
	}

	p := a.Runtime
	if a.Runtime == nil {
		p = a.CompileSpec.Package
	}

	if fileHelper.Exists(finalizeFile) {
		out, err := helpers.RenderFiles(
			helpers.ChartFile(finalizeFile),
			defFile,
		)
		if err != nil {
			Warning("Failed rendering finalizer for ",
				p.HumanReadableString(), err.Error())
			return err
		}

		Info("Executing finalizer for " + p.HumanReadableString())
		finalizer, err := NewLuetFinalizerFromYaml([]byte(out))
		if err != nil {
			Warning("Failed reading finalizer for ",
				p.HumanReadableString(), err.Error())
			return err
		}

		if postInstall {
			err = finalizer.RunInstall(targetRootfs)
		} else {
			err = finalizer.RunUninstall(targetRootfs)
		}
		if err != nil {
			Warning("Failed running finalizer for ",
				p.HumanReadableString(), err.Error())
			return err
		}
	}

	return nil
}

// NOTE: These methods will be replaced soon

func (m *ArtifactsManager) ExistsPackageFile(file string) (bool, *pkg.DefaultPackage, error) {
	Debug("Checking if file ", file, "belongs to any package")
	m.buildIndexFiles()
	m.Lock()
	defer m.Unlock()
	if p, exists := m.fileIndex[file]; exists {
		Debug(file, "belongs already to", p.HumanReadableString())

		return exists, p, nil
	}
	Debug(file, "doesn't belong to any package")
	return false, nil, nil
}

func (m *ArtifactsManager) buildIndexFiles() {
	m.Lock()
	defer m.Unlock()

	Debug("Building index files...")
	start := time.Now()

	// Check if cache is empty or if it got modified
	if m.fileIndex == nil { //|| len(s.Database.GetPackages()) != len(s.fileIndex) {
		m.fileIndex = make(map[string]*pkg.DefaultPackage)
		for _, p := range m.Database.World() {
			files, _ := m.Database.GetPackageFiles(p)
			for _, f := range files {
				m.fileIndex[f] = p.(*pkg.DefaultPackage)
			}
		}
	}

	Debug(fmt.Sprintf("Build index files completed in %d µs.",
		time.Now().Sub(start).Nanoseconds()/1e3))
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/geaaru/luet/pkg/config"
	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
	. "github.com/geaaru/luet/pkg/logger"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	"github.com/geaaru/luet/pkg/v2/repository/client"
	"github.com/pkg/errors"
)

const (
	REPOSITORY_METAFILE  = "repository.meta.yaml"
	REPOSITORY_SPECFILE  = "repository.yaml"
	TREE_TARBALL         = "tree.tar"
	COMPILERTREE_TARBALL = "compilertree.tar"

	REPOFILE_TREE_KEY          = "tree"
	REPOFILE_COMPILER_TREE_KEY = "compilertree"
	REPOFILE_META_KEY          = "meta"

	// TODO: To move on a specific package
	DiskRepositoryType   = "disk"
	HttpRepositoryType   = "http"
	DockerRepositoryType = "docker"
)

type Client interface {
	DownloadArtifact(*artifact.PackageArtifact) error
	DownloadFile(string) (string, error)
}

type WagonRepository struct {
	Identity *WagonIdentity
	Stones   *WagonStones
}

func NewWagonRepository(l *config.LuetRepository) *WagonRepository {
	return &WagonRepository{
		Identity: NewWagonIdentify(l),
		Stones:   NewWagonStones(),
	}
}

func (w *WagonRepository) SearchStones(opts *StonesSearchOpts) (*[]*Stone, error) {

	// Load catalog if not loaded yet
	if w.Stones.Catalog == nil {
		_, err := w.Stones.LoadCatalog(w.Identity)
		if err != nil {
			return nil, err
		}
	}

	return w.Stones.Search(opts, w.Identity.Name)
}

func (w *WagonRepository) SearchArtifacts(opts *StonesSearchOpts) (*[]*artifact.PackageArtifact, error) {

	// Load catalog if not loaded yet
	if w.Stones.Catalog == nil {
		_, err := w.Stones.LoadCatalog(w.Identity)
		if err != nil {
			return nil, err
		}
	}

	return w.Stones.SearchArtifacts(opts, w.Identity.Name)
}

func (w *WagonRepository) HasLocalWagonIdentity(wdir string) bool {
	file := filepath.Join(wdir, REPOSITORY_SPECFILE)
	if fileHelper.Exists(file) {
		return true
	}
	return false
}

func (w *WagonRepository) ReadWagonIdentify(wdir string) error {
	file := filepath.Join(wdir, REPOSITORY_SPECFILE)

	repoName := w.Identity.Name
	repoUrls := w.Identity.Urls
	repoPriority := w.Identity.Priority
	repoAuthentication := w.Identity.Authentication

	err := w.Identity.Load(file)
	if err != nil {
		return err
	}
	// Ensure that we use urls from config
	w.Identity.Name = repoName
	w.Identity.Urls = repoUrls
	w.Identity.Priority = repoPriority
	w.Identity.Authentication = repoAuthentication

	return nil
}

func (w *WagonRepository) GetRevision() int {
	return w.Identity.LuetRepository.Revision
}
func (w *WagonRepository) GetLastUpdate() string {
	return w.Identity.LuetRepository.LastUpdate
}
func (w *WagonRepository) SetLastUpdate(u string) {
	w.Identity.LuetRepository.LastUpdate = u
}
func (w *WagonRepository) IncrementRevision() {
	w.Identity.LuetRepository.Revision++
}

func (w *WagonRepository) ClearCatalog() {
	w.Stones.Catalog = nil
}

// The Sync method update the repository.
//
// In particular, follow these steps:
// * download the main repository.yaml file to a temporary directory
// * load the new repository.yaml as WagonIdentity and compare revision
//   and last update date with the current status.
// * if there is a new revision download the meta and tree file
//   and unpack them to the local cache.
//
// If force is true the download of the meta and tree files are done always.
func (w *WagonRepository) Sync(force bool) error {
	var treefs, metafs string
	aurora := GetAurora()

	Debug("Sync of the repository", w.Identity.Name, "in progress...")
	c := w.Client()
	if c == nil {
		return errors.New("no client could be generated from repository")
	}

	// Retrieve remote repository.yaml for retrieve revision and date
	file, err := c.DownloadFile(REPOSITORY_SPECFILE)
	if err != nil {
		return errors.Wrap(err, "While downloading "+REPOSITORY_SPECFILE)
	}

	repobasedir := config.LuetCfg.GetSystem().GetRepoDatabaseDirPath(w.Identity.Name)
	newIdentity := NewWagonIdentify(w.Identity.LuetRepository.Clone())
	err = newIdentity.Load(file)
	if err != nil {
		return err
	}

	if !newIdentity.Valid() {
		return errors.New("Corrupted remote repository.yaml file")
	}

	// Remove temporary file that contains repository.yaml
	// Example: /tmp/HttpClient236052003
	defer os.RemoveAll(file)

	toUpdate := w.Identity.Is2Update(newIdentity)

	if w.Identity.GetTreePath() == "" {
		treefs = filepath.Join(repobasedir, "treefs")
	} else {
		treefs = w.Identity.GetTreePath()
	}
	if w.Identity.GetMetaPath() == "" {
		metafs = filepath.Join(repobasedir, "metafs")
	} else {
		metafs = w.Identity.GetMetaPath()
	}

	newIdentity.LuetRepository.MetaPath = metafs
	newIdentity.LuetRepository.TreePath = treefs

	// treeFile and metaFile must be present, they aren't optional
	if toUpdate || force {

		treeFileArtifact, err := newIdentity.DownloadDocument(c, REPOFILE_TREE_KEY)
		if err != nil {
			return errors.Wrapf(err, "while fetching '%s'", REPOFILE_TREE_KEY)
		}
		defer os.Remove(treeFileArtifact.Path)

		Debug("Tree tarball for the repository " + w.Identity.GetName() + " downloaded correctly.")

		metaFileArtifact, err := newIdentity.DownloadDocument(c, REPOFILE_META_KEY)
		if err != nil {
			return errors.Wrapf(err, "while fetching '%s'", REPOFILE_META_KEY)
		}
		defer os.Remove(metaFileArtifact.Path)

		Debug("Metadata tarball for the repository " + w.Identity.GetName() + " downloaded correctly.")

		// Copy updated repository.yaml file to repo dir now that the tree is synced.
		err = fileHelper.CopyFile(file, filepath.Join(repobasedir, REPOSITORY_SPECFILE))
		if err != nil {
			return errors.Wrap(err, "Error on update "+REPOSITORY_SPECFILE)
		}
		// Remove previous tree
		os.RemoveAll(treefs)
		// Remove previous meta dir
		os.RemoveAll(metafs)

		Debug("Decompress tree of the repository " + w.Identity.GetName() + "...")

		err = treeFileArtifact.Unpack(treefs, false)
		if err != nil {
			return errors.Wrap(err, "Error met while unpacking tree")
		}

		// FIXME: It seems that tar with only one file doesn't create destination
		//       directory. I create directory directly for now.
		os.MkdirAll(metafs, os.ModePerm)
		err = metaFileArtifact.Unpack(metafs, false)
		if err != nil {
			return errors.Wrap(err, "Error met while unpacking metadata")
		}

		tsec, _ := strconv.ParseInt(newIdentity.GetLastUpdate(), 10, 64)

		InfoC(
			aurora.Bold(
				aurora.Red(fmt.Sprintf(
					":house:Repository: %30s Revision: ",
					w.Identity.GetName()))).String() +
				aurora.Bold(aurora.Green(fmt.Sprintf("%3d", newIdentity.GetRevision()))).String() + " - " +
				aurora.Bold(aurora.Green(time.Unix(tsec, 0).String())).String(),
		)

		w.Identity = newIdentity

	} else {
		InfoC(
			aurora.Magenta(":information_source: Repository: ").String() +
				aurora.Bold(
					aurora.Green(fmt.Sprintf("%30s", w.Identity.GetName())).String()+
						" is already up to date.",
				).String(),
		)
	}

	return nil
}

func (w *WagonRepository) GetTreePath(repobasedir string) string {
	if w.Identity.GetTreePath() == "" {

		// NOTE: repobasedir must be the value of
		//       LuetCfg.GetSystem().GetSystemReposDirPath
		repobase := filepath.Join(repobasedir, w.Identity.GetName())
		return filepath.Join(repobase, "treefs")
	}
	return w.Identity.GetTreePath()
}

func (w *WagonRepository) GetMetaPath(repobasedir string) string {
	if w.Identity.GetTreePath() == "" {
		// NOTE: repobasedir must be the value of
		//       LuetCfg.GetSystem().GetSystemReposDirPath
		repobase := filepath.Join(repobasedir, w.Identity.GetName())
		return filepath.Join(repobase, "metafs")
	}
	return w.Identity.GetTreePath()
}

func (w *WagonRepository) Client() Client {
	switch w.Identity.GetType() {
	case DiskRepositoryType:
		return client.NewLocalClient(w.Identity.LuetRepository)
	case HttpRepositoryType:
		return client.NewHttpClient(w.Identity.LuetRepository)
	case DockerRepositoryType:
		return client.NewDockerClient(w.Identity.LuetRepository)
	}
	return nil
}

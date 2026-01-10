/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package installer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	artifact "github.com/macaroni-os/anise/pkg/compiler/types/artifact"
	compression "github.com/macaroni-os/anise/pkg/compiler/types/compression"
	fileHelper "github.com/macaroni-os/anise/pkg/helpers/file"
	"go.uber.org/multierr"

	"github.com/macaroni-os/anise/anise-build/pkg/installer/client"
	"github.com/macaroni-os/anise/pkg/compiler"
	"github.com/macaroni-os/anise/pkg/config"
	. "github.com/macaroni-os/anise/pkg/logger"
	pkg "github.com/macaroni-os/anise/pkg/package"
	tree "github.com/macaroni-os/anise/pkg/tree"

	//"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	REPOSITORY_METAFILE  = "repository.meta.yaml"
	REPOSITORY_SPECFILE  = "repository.yaml"
	TREE_TARBALL         = "tree.tar"
	COMPILERTREE_TARBALL = "compilertree.tar"

	REPOFILE_TREE_KEY          = "tree"
	REPOFILE_COMPILER_TREE_KEY = "compilertree"
	REPOFILE_META_KEY          = "meta"

	DiskRepositoryType   = "disk"
	HttpRepositoryType   = "http"
	DockerRepositoryType = "docker"
)

type AniseRepositoryFile struct {
	FileName        string                     `json:"filename" yaml:"filename"`
	CompressionType compression.Implementation `json:"compressiontype,omitempty" yaml:"compressiontype,omitempty"`
	Checksums       artifact.Checksums         `json:"checksums,omitempty" yaml:"checksums,omitempty"`
}

type AniseSystemRepository struct {
	config.AniseRepository `yaml:",inline"`

	Index           compiler.ArtifactIndex         `json:"index" yaml:"index"`
	BuildTree, Tree tree.Builder                   `json:"-" yaml:"-"`
	RepositoryFiles map[string]AniseRepositoryFile `json:"repo_files" yaml:"repo_files"`
	Backend         compiler.CompilerBackend       `json:"-" yaml:"-"`
	PushImages      bool                           `json:"-" yaml:"-"`
	ForcePush       bool                           `json:"-" yaml:"-"`

	imagePrefix string `json:"-" yaml:"-"`
}

type AniseSystemRepositoryMetadata struct {
	Index []*artifact.PackageArtifact `json:"index,omitempty" yaml:"index,omitempty"`
}

type AniseSearchModeType int

const (
	SLabel      = iota
	SRegexPkg   = iota
	SRegexLabel = iota
	FileSearch  = iota
)

type AniseSearchOpts struct {
	Mode AniseSearchModeType
}

func NewAniseSystemRepositoryMetadata(file string, removeFile bool) (*AniseSystemRepositoryMetadata, error) {
	ans := &AniseSystemRepositoryMetadata{}
	err := ans.ReadFile(file, removeFile)
	if err != nil {
		return nil, err
	}
	return ans, nil
}

// SystemRepositories returns the repositories from the local configuration file
func SystemRepositories(c *config.AniseConfig) Repositories {
	repos := Repositories{}
	for _, repo := range c.SystemRepositories {
		if !repo.Enable {
			continue
		}
		r := NewSystemRepository(repo)
		repos = append(repos, r)
	}
	return repos
}

// LoadBuildTree loads to the tree the compilation specs from the system repositories
func LoadBuildTree(t tree.Builder, db pkg.PackageDatabase, c *config.AniseConfig) error {
	var reserr error
	repos := SystemRepositories(c)
	for _, r := range repos {
		repodir, err := config.AniseCfg.GetSystem().TempDir(r.Name)
		if err != nil {
			reserr = multierr.Append(reserr, err)
		}
		if err := r.SyncBuildMetadata(repodir); err != nil {
			reserr = multierr.Append(reserr, err)
		}

		generalRecipe := tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))
		if err := generalRecipe.Load(filepath.Join(repodir, "tree")); err != nil {
			reserr = multierr.Append(reserr, err)
		}
		if err := generalRecipe.GetDatabase().Clone(t.GetDatabase()); err != nil {
			reserr = multierr.Append(reserr, err)
		}

		r.SetTree(generalRecipe)
	}

	repos.SyncDatabase(db)

	return reserr
}

func (m *AniseSystemRepositoryMetadata) WriteFile(path string) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (m *AniseSystemRepositoryMetadata) ReadFile(file string, removeFile bool) error {
	if file == "" {
		return errors.New("Invalid path for repository metadata")
	}

	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if removeFile {
		defer os.Remove(file)
	}

	err = yaml.Unmarshal(dat, m)
	if err != nil {
		return err
	}

	return nil
}

func (m *AniseSystemRepositoryMetadata) ToArtifactIndex() (ans compiler.ArtifactIndex) {
	for _, a := range m.Index {
		ans = append(ans, a)
	}
	return
}

func NewDefaultTreeRepositoryFile() AniseRepositoryFile {
	return AniseRepositoryFile{
		FileName:        TREE_TARBALL,
		CompressionType: compression.GZip,
	}
}

func NewDefaultCompilerTreeRepositoryFile() AniseRepositoryFile {
	return AniseRepositoryFile{
		FileName:        COMPILERTREE_TARBALL,
		CompressionType: compression.GZip,
	}
}

func NewDefaultMetaRepositoryFile() AniseRepositoryFile {
	return AniseRepositoryFile{
		FileName:        REPOSITORY_METAFILE + ".tar",
		CompressionType: compression.None,
	}
}

// SetFileName sets the name of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this set the filename that the client will pull
func (f *AniseRepositoryFile) SetFileName(n string) {
	f.FileName = n
}

// GetFileName returns the name of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this gets the filename that the client will pull
func (f *AniseRepositoryFile) GetFileName() string {
	return f.FileName
}

// SetCompressionType sets the compression type of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this sets the compression type that the client will use to uncompress the artifact
func (f *AniseRepositoryFile) SetCompressionType(c compression.Implementation) {
	f.CompressionType = c
}

// GetCompressionType gets the compression type of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this gets the compression type that the client will use to uncompress the artifact
func (f *AniseRepositoryFile) GetCompressionType() compression.Implementation {
	return f.CompressionType
}

// SetChecksums sets the checksum of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this sets the checksums that the client will use to verify the artifact
func (f *AniseRepositoryFile) SetChecksums(c artifact.Checksums) {
	f.Checksums = c
}

// GetChecksums gets the checksum of the repository file.
// Each repository can ship arbitrary file that will be downloaded by the client
// in case of need, this gets the checksums that the client will use to verify the artifact
func (f *AniseRepositoryFile) GetChecksums() artifact.Checksums {
	return f.Checksums
}

// GenerateRepository generates a new repository from the given argument.
// If the repository is of the docker type, it will also push the package images.
// In case the repository is local, it will build the package Index
func GenerateRepository(p ...RepositoryOption) (*AniseSystemRepository, error) {
	c := RepositoryConfig{}
	c.Apply(p...)

	btr := tree.NewCompilerRecipe(pkg.NewInMemoryDatabase(false))
	runtimeTree := pkg.NewInMemoryDatabase(false)

	tempTree := pkg.NewInMemoryDatabase(false)
	temptr := tree.NewInstallerRecipe(tempTree)

	for _, treeDir := range c.Tree {
		if err := temptr.Load(treeDir); err != nil {
			return nil, err
		}
		if err := btr.Load(treeDir); err != nil {
			return nil, err
		}
	}

	// 2: if fromRepo, build a new tree like the compiler is doing and use it to source the above specs,
	// instead of local tree

	repodb := pkg.NewInMemoryDatabase(false)
	generalRecipe := tree.NewCompilerRecipe(repodb)

	if c.FromRepository {
		if err := LoadBuildTree(generalRecipe, repodb, c.config); err != nil {
			Warning("errors while loading trees from repositories", err.Error())
		}

		if err := repodb.Clone(tempTree); err != nil {
			Warning("errors while cloning trees from repositories", err.Error())
		}

	}

	// Pick only atoms in db which have a real metadata for runtime db (tr)
	for _, p := range tempTree.World() {
		if _, err := os.Stat(filepath.Join(c.Src, p.GetMetadataFilePath())); err == nil {
			runtimeTree.CreatePackage(p)
		}
	}

	// Load packages from metadata files if not present already.
	var ff = func(currentpath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Only those which are metadata
		if !strings.HasSuffix(info.Name(), pkg.PackageMetaSuffix) {
			return nil
		}

		dat, err := ioutil.ReadFile(currentpath)
		if err != nil {
			return nil
		}

		art, err := artifact.NewPackageArtifactFromYaml(dat)
		if err != nil {
			return nil
		}
		if _, err := runtimeTree.FindPackage(art.CompileSpec.Package); err != nil && art.CompileSpec.Package.Name != "" {
			Debug("Adding", art.CompileSpec.Package.HumanReadableString(), "from metadata file", currentpath)
			if art.Runtime != nil && art.Runtime.Name != "" {
				runtimeTree.CreatePackage(art.Runtime)
			} else {
				// We don't have runtime at this point. So we import the package as is
				r := []*pkg.DefaultPackage{}
				p := art.CompileSpec.Package.Clone()
				p.Requires(r)
				p.SetProvides(r)
				p.Conflicts(r)
				runtimeTree.CreatePackage(p)
			}
		}

		return nil
	}

	if c.FromMetadata {
		// Best effort
		filepath.Walk(c.Src, ff)
	}

	repo := &AniseSystemRepository{
		AniseRepository: *config.NewAniseRepository(c.Name, c.Type, c.Description, c.Urls, c.Priority, true, false),
		Tree:            tree.NewInstallerRecipe(runtimeTree),
		BuildTree:       btr,
		RepositoryFiles: map[string]AniseRepositoryFile{},
		PushImages:      c.PushImages,
		ForcePush:       c.Force,
		Backend:         c.CompilerBackend,
		imagePrefix:     c.ImagePrefix,
	}

	if err := repo.initialize(c.Src); err != nil {
		return nil, errors.Wrap(err, "while building repository artifact index")
	}

	return repo, nil
}

func NewSystemRepository(repo config.AniseRepository) *AniseSystemRepository {
	return &AniseSystemRepository{
		AniseRepository: repo,
		RepositoryFiles: map[string]AniseRepositoryFile{},
	}
}

func (r *AniseSystemRepository) String() string {
	ans := ""
	d, err := yaml.Marshal(r)
	if err == nil {
		ans = string(d)
	}
	return ans
}

func NewAniseSystemRepositoryFromYaml(data []byte, db pkg.PackageDatabase) (*AniseSystemRepository, error) {
	var p *AniseSystemRepository
	err := yaml.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}

	p.Tree = tree.NewInstallerRecipe(db)

	return p, err
}

func (r *AniseSystemRepository) SetPriority(n int) {
	r.AniseRepository.Priority = n
}

func (r *AniseSystemRepository) initialize(src string) error {
	generator, err := r.getGenerator()
	if err != nil {
		return errors.Wrap(err, "while constructing repository generator")
	}
	art, err := generator.Initialize(src, r.Tree.GetDatabase())
	if err != nil {
		return errors.Wrap(err, "while initializing repository generator")
	}
	// update the repository index
	r.Index = art
	return nil
}

// FileSearch search a pattern among the artifacts in a repository
func (r *AniseSystemRepository) FileSearch(pattern string) (pkg.Packages, error) {
	var matches pkg.Packages
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return matches, err
	}
ARTIFACT:
	for _, a := range r.GetIndex() {
		for _, f := range a.Files {
			if reg.MatchString(f) {
				matches = append(matches, a.CompileSpec.GetPackage())
				continue ARTIFACT
			}
		}
	}
	return matches, nil
}

func (r *AniseSystemRepository) GetName() string {
	return r.AniseRepository.Name
}
func (r *AniseSystemRepository) GetDescription() string {
	return r.AniseRepository.Description
}

func (r *AniseSystemRepository) GetAuthentication() map[string]string {
	return r.AniseRepository.Authentication
}

func (r *AniseSystemRepository) GetType() string {
	return r.AniseRepository.Type
}

func (r *AniseSystemRepository) SetType(p string) {
	r.AniseRepository.Type = p
}

func (r *AniseSystemRepository) GetVerify() bool {
	return r.AniseRepository.Verify
}

func (r *AniseSystemRepository) SetVerify(p bool) {
	r.AniseRepository.Verify = p
}

func (r *AniseSystemRepository) GetBackend() compiler.CompilerBackend {
	return r.Backend
}
func (r *AniseSystemRepository) SetBackend(b compiler.CompilerBackend) {
	r.Backend = b
}

func (r *AniseSystemRepository) SetName(p string) {
	r.AniseRepository.Name = p
}

func (r *AniseSystemRepository) AddUrl(p string) {
	r.AniseRepository.Urls = append(r.AniseRepository.Urls, p)
}
func (r *AniseSystemRepository) GetUrls() []string {
	return r.AniseRepository.Urls
}
func (r *AniseSystemRepository) SetUrls(urls []string) {
	r.AniseRepository.Urls = urls
}
func (r *AniseSystemRepository) GetPriority() int {
	return r.AniseRepository.Priority
}
func (r *AniseSystemRepository) GetTreePath() string {
	return r.TreePath
}
func (r *AniseSystemRepository) SetTreePath(p string) {
	r.TreePath = p
}
func (r *AniseSystemRepository) GetMetaPath() string {
	return r.MetaPath
}
func (r *AniseSystemRepository) SetMetaPath(p string) {
	r.MetaPath = p
}
func (r *AniseSystemRepository) SetTree(b tree.Builder) {
	r.Tree = b
}
func (r *AniseSystemRepository) GetIndex() compiler.ArtifactIndex {
	return r.Index
}
func (r *AniseSystemRepository) SetIndex(i compiler.ArtifactIndex) {
	r.Index = i
}
func (r *AniseSystemRepository) GetTree() tree.Builder {
	return r.Tree
}
func (r *AniseSystemRepository) GetRevision() int {
	return r.AniseRepository.Revision
}
func (r *AniseSystemRepository) GetLastUpdate() string {
	return r.AniseRepository.LastUpdate
}
func (r *AniseSystemRepository) SetLastUpdate(u string) {
	r.AniseRepository.LastUpdate = u
}
func (r *AniseSystemRepository) IncrementRevision() {
	r.AniseRepository.Revision++
}
func (r *AniseSystemRepository) SetAuthentication(auth map[string]string) {
	r.AniseRepository.Authentication = auth
}

// BumpRevision bumps the internal repository revision by reading the current one from repospec
func (r *AniseSystemRepository) BumpRevision(repospec string, resetRevision bool) error {
	if resetRevision {
		r.Revision = 0
	} else {
		if _, err := os.Stat(repospec); !os.IsNotExist(err) {
			// Read existing file for retrieve revision
			spec, err := r.ReadSpecFile(repospec)
			if err != nil {
				return err
			}
			r.Revision = spec.GetRevision()
		}
	}
	r.Revision++
	return nil
}

// AddMetadata adds the repository serialized content into the metadata key of the repository
// It writes the serialized content to repospec, and writes the repository.meta.yaml file into dst
func (r *AniseSystemRepository) AddMetadata(repospec, dst string) (*artifact.PackageArtifact, error) {
	// Create Metadata struct and serialized repository
	meta, serialized := r.Serialize()

	// Create metadata file and repository file
	metaTmpDir, err := config.AniseCfg.GetSystem().TempDir("metadata")
	defer os.RemoveAll(metaTmpDir) // clean up
	if err != nil {
		return nil, errors.Wrap(err, "Error met while creating tempdir for metadata")
	}

	repoMetaSpec := filepath.Join(metaTmpDir, REPOSITORY_METAFILE)

	// Create repository.meta.yaml file
	err = meta.WriteFile(repoMetaSpec)
	if err != nil {
		return nil, err
	}
	a, err := r.AddRepositoryFile(metaTmpDir, REPOFILE_META_KEY, dst, NewDefaultMetaRepositoryFile())
	if err != nil {
		return a, errors.Wrap(err, "Error met while adding archive to repository")
	}

	data, err := yaml.Marshal(serialized)
	if err != nil {
		return a, err
	}
	err = ioutil.WriteFile(repospec, data, os.ModePerm)
	if err != nil {
		return a, err
	}
	return a, nil
}

// AddTree adds a tree.Builder with the given key to the repository.
// It will generate an artifact which will be then embedded in the repository manifest
// It returns the generated artifacts and an error
func (r *AniseSystemRepository) AddTree(t tree.Builder, dst, key string, f AniseRepositoryFile) (*artifact.PackageArtifact, error) {
	// Create tree and repository file
	archive, err := config.AniseCfg.GetSystem().TempDir("archive")
	if err != nil {
		return nil, errors.Wrap(err, "Error met while creating tempdir for archive")
	}
	defer os.RemoveAll(archive) // clean up

	if err := t.Save(archive); err != nil {
		return nil, errors.Wrap(err, "Error met while saving the tree")
	}

	a, err := r.AddRepositoryFile(archive, key, dst, f)
	if err != nil {
		return nil, errors.Wrap(err, "Error met while adding archive to repository")
	}
	return a, nil
}

// AddRepositoryFile adds a path to a key in the repository manifest.
// The path will be compressed, and a default File has to be passed in case there is no entry into
// the repository manifest
func (r *AniseSystemRepository) AddRepositoryFile(src, fileKey, repositoryRoot string, defaults AniseRepositoryFile) (*artifact.PackageArtifact, error) {
	treeFile, err := r.GetRepositoryFile(fileKey)
	if err != nil {
		treeFile = defaults
		//	r.SetRepositoryFile(fileKey, treeFile)
	}

	a := artifact.NewPackageArtifact(filepath.Join(repositoryRoot, treeFile.GetFileName()))
	a.CompressionType = treeFile.GetCompressionType()
	err = a.Compress(src, 1)
	if err != nil {
		return a, errors.Wrap(err, "Error met while creating package archive")
	}

	err = a.Hash()
	if err != nil {
		return a, errors.Wrap(err, "Failed generating checksums for tree")
	}
	// Update the tree name with the name created by compression selected.
	treeFile.SetChecksums(a.Checksums)
	treeFile.SetFileName(path.Base(a.Path))

	r.SetRepositoryFile(fileKey, treeFile)

	return a, nil
}

func (r *AniseSystemRepository) GetRepositoryFile(name string) (AniseRepositoryFile, error) {
	ans, ok := r.RepositoryFiles[name]
	if ok {
		return ans, nil
	}
	return ans, errors.New("Repository file " + name + " not found!")
}
func (r *AniseSystemRepository) SetRepositoryFile(name string, f AniseRepositoryFile) {
	r.RepositoryFiles[name] = f
}

func (r *AniseSystemRepository) ReadSpecFile(file string) (*AniseSystemRepository, error) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading file "+file)
	}

	var repo *AniseSystemRepository
	repo, err = NewAniseSystemRepositoryFromYaml(dat, pkg.NewInMemoryDatabase(false))
	if err != nil {
		return nil, errors.Wrap(err, "Error reading repository from file "+file)
	}

	// Check if mandatory key are present
	_, err = repo.GetRepositoryFile(REPOFILE_TREE_KEY)
	if err != nil {
		return nil, errors.New("Invalid repository without the " + REPOFILE_TREE_KEY + " key file.")
	}
	_, err = repo.GetRepositoryFile(REPOFILE_META_KEY)
	if err != nil {
		return nil, errors.New("Invalid repository without the " + REPOFILE_META_KEY + " key file.")
	}

	return repo, err
}

type RepositoryGenerator interface {
	Generate(*AniseSystemRepository, string, bool) error
	Initialize(string, pkg.PackageDatabase) ([]*artifact.PackageArtifact, error)
}

func (r *AniseSystemRepository) getGenerator() (RepositoryGenerator, error) {
	var rg RepositoryGenerator
	switch r.GetType() {
	case DiskRepositoryType, HttpRepositoryType:
		rg = &localRepositoryGenerator{}
	case DockerRepositoryType:
		rg = &dockerRepositoryGenerator{
			b:           r.Backend,
			imagePrefix: r.imagePrefix,
			imagePush:   r.PushImages,
			force:       r.ForcePush,
		}
	default:
		return nil, errors.New("invalid repository type")
	}
	return rg, nil
}

// Write writes the repository metadata to the supplied destination
func (r *AniseSystemRepository) Write(dst string, resetRevision, force bool) error {
	rg, err := r.getGenerator()
	if err != nil {
		return err
	}

	return rg.Generate(r, dst, resetRevision)
}

func (r *AniseSystemRepository) Client() Client {
	switch r.GetType() {
	case DiskRepositoryType:
		return client.NewLocalClient(client.RepoData{Urls: r.GetUrls()})
	case HttpRepositoryType:
		return client.NewHttpClient(
			client.RepoData{
				Urls:           r.GetUrls(),
				Authentication: r.GetAuthentication(),
			})

	case DockerRepositoryType:
		return client.NewDockerClient(
			client.RepoData{
				Urls:           r.GetUrls(),
				Authentication: r.GetAuthentication(),
				Verify:         r.Verify,
			})
	}
	return nil
}

func (r *AniseSystemRepository) SearchArtefact(p pkg.Package) (*artifact.PackageArtifact, error) {
	for _, a := range r.GetIndex() {
		if a.CompileSpec.GetPackage().Matches(p) {
			return a, nil
		}
	}

	return nil, errors.New("Not found")
}

func (r *AniseSystemRepository) getRepoFile(c Client, key string) (*artifact.PackageArtifact, error) {

	treeFile, err := r.GetRepositoryFile(key)
	if err != nil {
		return nil, errors.Wrapf(err, "key %s not present in the repository", key)
	}

	// Get Tree
	downloadedTreeFile, err := c.DownloadFile(treeFile.GetFileName())
	if err != nil {
		return nil, errors.Wrap(err, "While downloading "+treeFile.GetFileName())
	}
	//defer os.Remove(downloadedTreeFile)

	treeFileArtifact := artifact.NewPackageArtifact(downloadedTreeFile)
	treeFileArtifact.Checksums = treeFile.GetChecksums()
	treeFileArtifact.CompressionType = treeFile.GetCompressionType()

	err = treeFileArtifact.Verify()
	if err != nil {
		return nil, errors.Wrap(err, "file integrity check failure")
	}

	return treeFileArtifact, nil

}

func (r *AniseSystemRepository) SyncBuildMetadata(path string) error {

	repo, err := r.Sync(false)
	if err != nil {
		return errors.Wrap(err, "while syncronizing repository")
	}

	c := repo.Client()
	if c == nil {
		return errors.New("no client could be generated from repository")
	}

	a, err := repo.getRepoFile(c, REPOFILE_COMPILER_TREE_KEY)
	if err != nil {
		return fmt.Errorf("failed while getting: %s", REPOFILE_COMPILER_TREE_KEY)
	}

	defer os.RemoveAll(a.Path)

	if err := a.Unpack(filepath.Join(path, "tree"), false); err != nil {
		return errors.Wrapf(err, "while unpacking: %s", REPOFILE_COMPILER_TREE_KEY)
	}

	for _, ai := range repo.GetTree().GetDatabase().World() {
		// Retrieve remote repository.yaml for retrieve revision and date
		file, err := c.DownloadFile(ai.GetMetadataFilePath())
		if err != nil {
			return errors.Wrapf(err, "while downloading metadata for %s", ai.HumanReadableString())
		}
		if err := fileHelper.Move(file, filepath.Join(path, ai.GetMetadataFilePath())); err != nil {
			return err
		}
	}

	return nil
}

func (r *AniseSystemRepository) Load(alternativeRepoSpecfile, alternativeTreeFs, alternativeMetafs string) (*AniseSystemRepository, error) {
	var treefs, metafs string
	aurora := GetAurora()

	repobasedir := config.AniseCfg.GetSystem().GetRepoDatabaseDirPath(r.GetName())

	repospecfile := filepath.Join(repobasedir, REPOSITORY_SPECFILE)
	if alternativeRepoSpecfile != "" {
		repospecfile = alternativeRepoSpecfile
	}

	if alternativeTreeFs != "" {
		treefs = alternativeTreeFs
	} else {
		if r.GetTreePath() == "" {
			treefs = filepath.Join(repobasedir, "treefs")
		} else {
			treefs = r.GetTreePath()
		}
	}

	if alternativeMetafs != "" {
		metafs = alternativeMetafs
	} else {
		if r.GetMetaPath() == "" {
			metafs = filepath.Join(repobasedir, "metafs")
		} else {
			metafs = r.GetMetaPath()
		}
	}
	metafile := filepath.Join(metafs, REPOSITORY_METAFILE)

	Debug(fmt.Sprintf("[%s] Using spec file %s", r.GetName(), repospecfile))

	if !fileHelper.Exists(repospecfile) || !fileHelper.Exists(metafile) ||
		!fileHelper.Exists(treefs) {
		return nil, errors.New(
			fmt.Sprintf("The repository %s is not available.\n"+
				"You need to sync the database.", r.GetName()))
	}

	repoMeta, err := r.ReadSpecFile(repospecfile)
	if err != nil {
		return nil, err
	}

	meta, err := NewAniseSystemRepositoryMetadata(metafile, false)
	if err != nil {
		return nil, errors.Wrap(err, "While processing "+REPOSITORY_METAFILE)
	}
	repoMeta.SetIndex(meta.ToArtifactIndex())

	reciper := tree.NewInstallerRecipe(pkg.NewInMemoryDatabase(false))
	err = reciper.Load(treefs)
	if err != nil {
		return nil, errors.Wrap(err, "Error met while unpacking rootfs")
	}

	repoMeta.SetTree(reciper)
	repoMeta.SetTreePath(treefs)

	// Copy the local available data to the one which was synced
	// e.g. locally we can override the type (disk), or priority
	// while remotely it could be advertized differently
	r.fill(repoMeta)

	InfoC(
		aurora.Yellow(":information_source: ").String() +
			aurora.Magenta("Repository: ").String() +
			aurora.Green(aurora.Bold(fmt.Sprintf("%30s", repoMeta.GetName())).String()).String() +
			aurora.Magenta(" Priority: ").String() +
			aurora.Bold(aurora.Green(fmt.Sprintf("%3d", repoMeta.GetPriority()))).String() +
			aurora.Magenta(" Type: ").String() +
			aurora.Bold(aurora.Green(fmt.Sprintf("%5s", repoMeta.GetType()))).String() +
			aurora.Magenta(" Revision: ").String() +
			aurora.Bold(aurora.Green(fmt.Sprintf("%3d", repoMeta.GetRevision()))).String(),
	)
	return repoMeta, nil
}

func (r *AniseSystemRepository) Sync(force bool) (*AniseSystemRepository, error) {
	var repoUpdated bool = false
	var treefs, metafs string
	aurora := GetAurora()

	Debug("Sync of the repository", r.Name, "in progress...")
	c := r.Client()
	if c == nil {
		return nil, errors.New("no client could be generated from repository")
	}

	// Retrieve remote repository.yaml for retrieve revision and date
	file, err := c.DownloadFile(REPOSITORY_SPECFILE)
	if err != nil {
		return nil, errors.Wrap(err, "While downloading "+REPOSITORY_SPECFILE)
	}

	repobasedir := config.AniseCfg.GetSystem().GetRepoDatabaseDirPath(r.GetName())
	downloadedRepoMeta, err := r.ReadSpecFile(file)
	if err != nil {
		return nil, err
	}
	// Remove temporary file that contains repository.yaml
	// Example: /tmp/HttpClient236052003
	defer os.RemoveAll(file)

	if r.Cached {
		if !force {
			localRepo, _ := r.ReadSpecFile(filepath.Join(repobasedir, REPOSITORY_SPECFILE))
			if localRepo != nil {
				if localRepo.GetRevision() == downloadedRepoMeta.GetRevision() &&
					localRepo.GetLastUpdate() == downloadedRepoMeta.GetLastUpdate() {
					repoUpdated = true
				}
			}
		}
		if r.GetTreePath() == "" {
			treefs = filepath.Join(repobasedir, "treefs")
		} else {
			treefs = r.GetTreePath()
		}
		if r.GetMetaPath() == "" {
			metafs = filepath.Join(repobasedir, "metafs")
		} else {
			metafs = r.GetMetaPath()
		}

	} else {
		treefs, err = config.AniseCfg.GetSystem().TempDir("treefs")
		if err != nil {
			return nil, errors.Wrap(err, "Error met while creating tempdir for rootfs")
		}
		metafs, err = config.AniseCfg.GetSystem().TempDir("metafs")
		if err != nil {
			return nil, errors.Wrap(err, "Error met whilte creating tempdir for metafs")
		}
	}

	// treeFile and metaFile must be present, they aren't optional
	if !repoUpdated {

		treeFileArtifact, err := downloadedRepoMeta.getRepoFile(c, REPOFILE_TREE_KEY)
		if err != nil {
			return nil, errors.Wrapf(err, "while fetching '%s'", REPOFILE_TREE_KEY)
		}
		defer os.Remove(treeFileArtifact.Path)

		Debug("Tree tarball for the repository " + r.GetName() + " downloaded correctly.")

		metaFileArtifact, err := downloadedRepoMeta.getRepoFile(c, REPOFILE_META_KEY)
		if err != nil {
			return nil, errors.Wrapf(err, "while fetching '%s'", REPOFILE_META_KEY)
		}
		defer os.Remove(metaFileArtifact.Path)

		Debug("Metadata tarball for the repository " + r.GetName() + " downloaded correctly.")

		if r.Cached {
			// Copy updated repository.yaml file to repo dir now that the tree is synced.
			err = fileHelper.CopyFile(file, filepath.Join(repobasedir, REPOSITORY_SPECFILE))
			if err != nil {
				return nil, errors.Wrap(err, "Error on update "+REPOSITORY_SPECFILE)
			}
			// Remove previous tree
			os.RemoveAll(treefs)
			// Remove previous meta dir
			os.RemoveAll(metafs)
		}
		Debug("Decompress tree of the repository " + r.Name + "...")

		err = treeFileArtifact.Unpack(treefs, false)
		if err != nil {
			return nil, errors.Wrap(err, "Error met while unpacking tree")
		}

		// FIXME: It seems that tar with only one file doesn't create destination
		//       directory. I create directory directly for now.
		os.MkdirAll(metafs, os.ModePerm)
		err = metaFileArtifact.Unpack(metafs, false)
		if err != nil {
			return nil, errors.Wrap(err, "Error met while unpacking metadata")
		}

		tsec, _ := strconv.ParseInt(downloadedRepoMeta.GetLastUpdate(), 10, 64)

		InfoC(
			aurora.Bold(
				aurora.Red(fmt.Sprintf(
					":house:Repository: %20s Revision: ",
					downloadedRepoMeta.GetName()))).String() +
				aurora.Bold(aurora.Green(fmt.Sprintf("%3d", downloadedRepoMeta.GetRevision()))).String() + " - " +
				aurora.Bold(aurora.Green(time.Unix(tsec, 0).String())).String(),
		)

	} else {
		InfoC(
			aurora.Magenta(":information_source: Repository: ").String() +
				aurora.Bold(
					aurora.Green(fmt.Sprintf("%30s", downloadedRepoMeta.GetName())).String()+
						" is already up to date.",
				).String(),
		)
	}

	if r.Cached {
		return r.Load("", "", "")
	} else {
		return r.Load(file, treefs, metafs)
	}
}

func (r *AniseSystemRepository) fill(r2 *AniseSystemRepository) {
	r2.SetUrls(r.GetUrls())
	r2.SetAuthentication(r.GetAuthentication())
	r2.SetType(r.GetType())
	r2.SetPriority(r.GetPriority())
	r2.SetName(r.GetName())
	r2.SetVerify(r.GetVerify())
}

func (r *AniseSystemRepository) Serialize() (*AniseSystemRepositoryMetadata, AniseSystemRepository) {

	serialized := *r
	serialized.Authentication = nil

	serialized.Index = compiler.ArtifactIndex{}

	meta := &AniseSystemRepositoryMetadata{
		Index: []*artifact.PackageArtifact{},
	}
	for _, a := range r.Index {
		cp := *a
		copy := &cp
		copy.Path = filepath.Base(copy.Path)
		meta.Index = append(meta.Index, copy)
	}

	return meta, serialized
}

func (r Repositories) Len() int      { return len(r) }
func (r Repositories) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r Repositories) Less(i, j int) bool {
	return r[i].GetPriority() < r[j].GetPriority()
}

func (r Repositories) World() pkg.Packages {
	cache := map[string]pkg.Package{}
	world := pkg.Packages{}

	// Get Uniques. Walk in reverse so the definitions of most prio-repo overwrites lower ones
	// In this way, when we will walk again later the deps sorting them by most higher prio we have better chance of success.
	for i := len(r) - 1; i >= 0; i-- {
		for _, p := range r[i].GetTree().GetDatabase().World() {
			cache[p.GetFingerPrint()] = p
		}
	}

	for _, v := range cache {
		world = append(world, v)
	}

	return world
}

func (r Repositories) SyncDatabase(d pkg.PackageDatabase) {
	cache := map[string]bool{}

	// Get Uniques. Walk in reverse so the definitions of most prio-repo overwrites lower ones
	// In this way, when we will walk again later the deps sorting them by most higher prio we have better chance of success.
	for i := len(r) - 1; i >= 0; i-- {
		for _, p := range r[i].GetTree().GetDatabase().World() {
			if _, ok := cache[p.GetFingerPrint()]; !ok {
				cache[p.GetFingerPrint()] = true
				d.CreatePackage(p)
			}
		}
	}
}

type PackageMatch struct {
	Repo     *AniseSystemRepository
	Artifact *artifact.PackageArtifact
	Package  pkg.Package
}

func (re Repositories) PackageMatches(p pkg.Packages) []PackageMatch {
	// TODO: Better heuristic. here we pick the first repo that contains the atom, sorted by priority but
	// we should do a permutations and get the best match, and in case there are more solutions the user should be able to pick
	sort.Sort(re)

	var matches []PackageMatch
PACKAGE:
	for _, pack := range p {
		for _, r := range re {
			c, err := r.GetTree().GetDatabase().FindPackage(pack)
			if err == nil {
				a, _ := r.SearchArtefact(pack)
				matches = append(matches, PackageMatch{Package: c, Repo: r, Artifact: a})
				continue PACKAGE
			}
		}
	}

	return matches

}

func (re Repositories) ResolveSelectors(p pkg.Packages) pkg.Packages {
	// If a selector is given, get the best from each repo
	sort.Sort(re) // respect prio
	var matches pkg.Packages
PACKAGE:
	for _, pack := range p {
	REPOSITORY:
		for _, r := range re {
			if pack.IsSelector() {
				c, err := r.GetTree().GetDatabase().FindPackageCandidate(pack)
				// If FindPackageCandidate returns the same package, it means it couldn't find one.
				// Skip this repository and keep looking.
				if err != nil { //c.String() == pack.String() {
					continue REPOSITORY
				}
				matches = append(matches, c)
				continue PACKAGE
			} else {
				// If it's not a selector, just append it
				matches = append(matches, pack)
			}
		}
	}

	return matches

}

func (re Repositories) SearchPackages(p string, t AniseSearchModeType) []PackageMatch {
	sort.Sort(re)
	var matches []PackageMatch
	var err error

	for _, r := range re {
		var repoMatches pkg.Packages

		switch t {
		case SRegexPkg:
			repoMatches, err = r.GetTree().GetDatabase().FindPackageMatch(p)
		case SLabel:
			repoMatches, err = r.GetTree().GetDatabase().FindPackageLabel(p)
		case SRegexLabel:
			repoMatches, err = r.GetTree().GetDatabase().FindPackageLabelMatch(p)
		case FileSearch:
			repoMatches, err = r.FileSearch(p)
		}

		if err == nil && len(repoMatches) > 0 {
			for _, pack := range repoMatches {
				a, _ := r.SearchArtefact(pack)
				matches = append(matches, PackageMatch{Package: pack, Repo: r, Artifact: a})
			}
		}
	}

	return matches
}

func (re Repositories) SearchLabelMatch(s string) []PackageMatch {
	return re.SearchPackages(s, SRegexLabel)
}

func (re Repositories) SearchLabel(s string) []PackageMatch {
	return re.SearchPackages(s, SLabel)
}

func (re Repositories) Search(s string) []PackageMatch {
	return re.SearchPackages(s, SRegexPkg)
}

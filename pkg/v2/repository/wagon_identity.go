/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"fmt"
	"io/ioutil"

	artifact "github.com/geaaru/luet/pkg/compiler/types/artifact"
	compression "github.com/geaaru/luet/pkg/compiler/types/compression"
	"github.com/geaaru/luet/pkg/config"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type WagonDocument struct {
	FileName        string                     `json:"filename" yaml:"filename"`
	CompressionType compression.Implementation `json:"compressiontype,omitempty" yaml:"compressiontype,omitempty"`
	Checksums       artifact.Checksums         `json:"checksums,omitempty" yaml:"checksums,omitempty"`
}

type WagonIdentity struct {
	*config.LuetRepository `yaml:",inline" json:",inline"`

	IdentityFile    string                    `yaml:-" json:"-"`
	RepositoryFiles map[string]*WagonDocument `yaml:"repo_files,omitempty" json:"repo_files,omitempty"`
}

func NewWagonIdentify(l *config.LuetRepository) *WagonIdentity {
	return &WagonIdentity{
		LuetRepository: l,
	}
}

func (w *WagonIdentity) Valid() bool {
	_, hasMeta := w.RepositoryFiles[REPOFILE_META_KEY]
	if !hasMeta {
		return false
	}

	_, hasTree := w.RepositoryFiles[REPOFILE_TREE_KEY]
	if !hasTree {
		return false
	}

	return true
}

func (w *WagonIdentity) GetTreePath() string   { return w.LuetRepository.TreePath }
func (w *WagonIdentity) GetMetaPath() string   { return w.LuetRepository.MetaPath }
func (w *WagonIdentity) GetName() string       { return w.LuetRepository.Name }
func (w *WagonIdentity) GetLastUpdate() string { return w.LuetRepository.LastUpdate }
func (w *WagonIdentity) GetRevision() int      { return w.LuetRepository.Revision }
func (w *WagonIdentity) GetType() string       { return w.LuetRepository.Type }
func (w *WagonIdentity) GetVerify() bool       { return w.LuetRepository.Verify }
func (w *WagonIdentity) GetUrls() []string     { return w.LuetRepository.Urls }

func (w *WagonIdentity) SetLastUpdate(u string) { w.LuetRepository.LastUpdate = u }
func (w *WagonIdentity) SetType(p string)       { w.LuetRepository.Type = p }
func (w *WagonIdentity) SetVerify(p bool)       { w.LuetRepository.Verify = p }

func (w *WagonIdentity) GetAuthentication() map[string]string {
	return w.LuetRepository.Authentication
}

func (w *WagonIdentity) IncrementRevision() {
	w.LuetRepository.Revision++
}

func (w *WagonIdentity) Load(f string) error {
	//previousName := w.LuetRepository.Name

	data, err := ioutil.ReadFile(f)
	if err != nil {
		return errors.Wrap(err, "Error on reading file "+f)
	}

	err = yaml.Unmarshal(data, w)
	if err != nil {
		return err
	}

	w.IdentityFile = f

	return nil
}

func (w *WagonIdentity) Is2Update(newIdentity *WagonIdentity) bool {
	ans := true

	if w.GetRevision() == newIdentity.GetRevision() &&
		w.GetLastUpdate() == newIdentity.GetLastUpdate() {
		ans = false
	}

	return ans
}

func (w *WagonIdentity) DownloadDocument(c Client, key string) (*artifact.PackageArtifact, error) {
	docFile, ok := w.RepositoryFiles[key]
	if !ok {
		return nil, errors.New(fmt.Sprintf("key %s not present in the repository", key))
	}

	downloadedFile, err := c.DownloadFile(docFile.GetFileName())
	if err != nil {
		return nil, errors.Wrap(err, "While downloading "+docFile.GetFileName())
	}

	docArtifact := artifact.NewPackageArtifact(downloadedFile)
	docArtifact.Checksums = docFile.GetChecksums()
	docArtifact.CompressionType = docFile.GetCompressionType()

	err = docArtifact.Verify()
	if err != nil {
		return nil, errors.Wrap(err, "file integrity check failure")
	}

	return docArtifact, nil
}

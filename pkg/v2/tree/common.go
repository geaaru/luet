/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	pkg "github.com/geaaru/luet/pkg/package"

	zstd "github.com/klauspost/compress/zstd"
	"gopkg.in/yaml.v3"
)

const (
	IDX_FILE = ".anise-idx.json.zstd"
)

type TreeIdx struct {
	Map map[string][]*TreeIdxPkg `json:"packages,omitempty" yaml:"packages,omitempty"`

	TreePath string `json:"-" yaml:"-"`
}

type GenOpts struct {
	DryRun bool
}

type TreeIdxPkg struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
}

func NewTreeIdx(tpath string) *TreeIdx {
	return &TreeIdx{
		Map:      make(map[string][]*TreeIdxPkg, 0),
		TreePath: tpath,
	}
}

func (t *TreeIdx) AddPackage(name string, p *TreeIdxPkg) {
	if v, ok := t.Map[name]; ok {
		t.Map[name] = append(v, p)
	} else {
		t.Map[name] = []*TreeIdxPkg{p}
	}
}

func (t *TreeIdx) GetPackageVersions(name string) ([]*TreeIdxPkg, bool) {
	val, ok := t.Map[name]
	return val, ok
}

func (t *TreeIdx) Write() error {
	fpath := filepath.Join(t.TreePath, IDX_FILE)
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(data)

	dst, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer dst.Close()

	enc, err := zstd.NewWriter(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(enc, buffer)
	if err != nil {
		enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	return nil
}

func (t *TreeIdx) ToYAML() ([]byte, error) {
	return yaml.Marshal(t)
}

func (t *TreeIdx) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TreeIdx) Merge(tI *TreeIdx) {
	for k, v := range tI.Map {
		for _, tp := range v {
			t.AddPackage(k, tp)
		}
	}
}

func (t *TreeIdx) Generate(treeDir string, opts *GenOpts) error {
	tm, err := t.generateIdxDir(treeDir, opts)
	if err != nil {
		return err
	}

	t.Merge(tm)
	return nil
}

func (t *TreeIdx) generateIdxDir(dir string, opts *GenOpts) (*TreeIdx, error) {
	ans := NewTreeIdx(dir)

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.New("Error on readdir " + dir + ":" + err.Error())
	}

	for _, file := range dirEntries {
		f := filepath.Join(dir, file.Name())
		if file.IsDir() {
			tChildren, err := t.generateIdxDir(f, opts)
			if err != nil {
				return nil, err
			}

			ans.Merge(tChildren)
		} else if file.Name() == pkg.PackageDefinitionFile {

			dp, err := ReadDefinitionFile(f)
			if err != nil {
				return nil, err
			}

			ans.AddPackage(dp.PackageName(), &TreeIdxPkg{
				Version: dp.GetVersion(),
				Path:    f,
			})

		} else if file.Name() == pkg.PackageCollectionFile {

			c, err := ReadCollectionFile(f)
			if err != nil {
				return nil, err
			}

			for _, p := range c.Packages {
				ans.AddPackage(p.PackageName(), &TreeIdxPkg{
					Version: p.GetVersion(),
					Path:    f,
				})
			}
		}

	}

	if !opts.DryRun {
		// Write index file.
		err = ans.Write()
		if err != nil {
			return nil, err
		}
	}

	return ans, nil
}

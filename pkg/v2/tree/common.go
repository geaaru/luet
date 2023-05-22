/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fileHelper "github.com/geaaru/luet/pkg/helpers/file"
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

func (t *TreeIdx) Read(treeDir string) error {
	// I consider that the tree will be
	// with the index file of the upper directory
	// that contains the packages of all sub-directories.

	idxfile := filepath.Join(treeDir, IDX_FILE)

	if !fileHelper.Exists(idxfile) {
		return errors.New(fmt.Sprintf("File %s doesn't exists",
			idxfile))
	}

	idxf, err := os.Open(idxfile)
	if err != nil {
		return err
	}
	defer idxf.Close()

	d, err := zstd.NewReader(idxf)
	if err != nil {
		return err
	}
	defer d.Close()

	data, err := io.ReadAll(d)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, t)
	if err != nil {
		return err
	}

	return nil
}

func (t *TreeIdx) Generate(treeDir string, opts *GenOpts) error {
	if treeDir == "" {
		return errors.New("Invalid tree directory")
	}

	if treeDir[len(treeDir)-1:len(treeDir)] == "/" {
		treeDir = treeDir[0 : len(treeDir)-1]
	}

	base := filepath.Dir(treeDir)
	tm, err := t.generateIdxDir(treeDir, base, opts)
	if err != nil {
		return err
	}

	t.Merge(tm)
	return nil
}

func (t *TreeIdx) generateIdxDir(dir, base string, opts *GenOpts) (*TreeIdx, error) {
	ans := NewTreeIdx(dir)

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.New("Error on readdir " + dir + ":" + err.Error())
	}

	for _, file := range dirEntries {
		f := filepath.Join(dir, file.Name())
		if file.IsDir() {
			tChildren, err := t.generateIdxDir(f, base, opts)
			if err != nil {
				return nil, err
			}

			ans.Merge(tChildren)
		} else if file.Name() == pkg.PackageDefinitionFile {

			dp, err := ReadDefinitionFile(f)
			if err != nil {
				return nil, err
			}

			relf, _ := filepath.Rel(base, f)

			ans.AddPackage(dp.PackageName(), &TreeIdxPkg{
				Version: dp.GetVersion(),
				Path:    relf,
			})

		} else if file.Name() == pkg.PackageCollectionFile {

			c, err := ReadCollectionFile(f)
			if err != nil {
				return nil, err
			}

			relf, _ := filepath.Rel(base, f)

			for _, p := range c.Packages {
				ans.AddPackage(p.PackageName(), &TreeIdxPkg{
					Version: p.GetVersion(),
					Path:    relf,
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

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
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"

	zstd "github.com/klauspost/compress/zstd"
	"gopkg.in/yaml.v3"
)

const (
	IDX_FILE = ".anise-idx.json"
)

type TreeIdx struct {
	Map     map[string][]*TreeIdxPkg `json:"packages,omitempty" yaml:"packages,omitempty"`
	BaseDir string                   `json:"basedir,omitempty" yaml:"basedir,omitempty"`

	Compress bool   `json:"-" yaml:"-"`
	TreePath string `json:"-" yaml:"-"`
}

type GenOpts struct {
	DryRun   bool
	OnlyMain bool
}

type TreeIdxPkg struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
}

func NewTreeIdx(tpath string, compress bool) *TreeIdx {
	return &TreeIdx{
		Map:      make(map[string][]*TreeIdxPkg, 0),
		TreePath: tpath,
		Compress: compress,
	}
}

func (t *TreeIdx) DetectMode() *TreeIdx {
	f := filepath.Join(t.TreePath, IDX_FILE)
	if fileHelper.Exists(f) {
		t.Compress = false
	} else if fileHelper.Exists(f + ".zstd") {
		t.Compress = true
	}

	return t
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

func (t *TreeIdx) GetPackageVersion(name, version string) (*TreeIdxPkg, bool) {
	val, ok := t.Map[name]
	for _, v := range val {
		if v.Version == version {
			return v, ok
		}
	}
	return nil, false
}

func (t *TreeIdx) Write() error {
	fpath := filepath.Join(t.TreePath, IDX_FILE)
	if t.Compress {
		fpath += ".zstd"
	}

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

	if t.Compress {
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
	} else {
		_, err = io.Copy(dst, buffer)
		if err != nil {
			return err
		}
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
	if t.Compress {
		idxfile += ".zstd"
	}

	if !fileHelper.Exists(idxfile) {
		return errors.New(fmt.Sprintf("File %s doesn't exists",
			idxfile))
	}

	idxf, err := os.Open(idxfile)
	if err != nil {
		return err
	}
	defer idxf.Close()

	var reader io.Reader
	if t.Compress {
		d, err := zstd.NewReader(idxf)
		if err != nil {
			return err
		}
		defer d.Close()

		reader = d
	} else {
		reader = idxf
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, t)
	if err != nil {
		return err
	}

	t.TreePath = treeDir

	return nil
}

func (t *TreeIdx) Generate(treeDir string, opts *GenOpts) error {
	var err error

	if treeDir == "" {
		return errors.New("Invalid tree directory")
	}

	if treeDir[len(treeDir)-1:len(treeDir)] == "/" {
		treeDir = treeDir[0 : len(treeDir)-1]
	}

	if !filepath.IsAbs(treeDir) {
		treeDir, err = filepath.Abs(treeDir)
		if err != nil {
			return err
		}
	}
	base := filepath.Dir(treeDir)

	Debug(fmt.Sprintf("Generating tree %s (base %s)...", treeDir, base))

	tm, err := t.generateIdxDir(treeDir, base, opts)
	if err != nil {
		return err
	}

	t.Merge(tm)

	t.BaseDir, _ = filepath.Rel(treeDir, base)
	return nil
}

func (t *TreeIdx) generateIdxDir(dir, base string, opts *GenOpts) (*TreeIdx, error) {
	ans := NewTreeIdx(dir, t.Compress)

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

			relf, err := filepath.Rel(base, f)
			if err != nil {
				return nil, err
			}

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
		ans.BaseDir, _ = filepath.Rel(dir, base)

		if !opts.OnlyMain || (opts.OnlyMain && ans.BaseDir == "..") {
			// Write index file.
			err = ans.Write()
			if err != nil {
				return nil, err
			}
		}
	}

	return ans, nil
}

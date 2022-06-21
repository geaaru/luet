/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"io/ioutil"

	artifact "github.com/geaaru/luet/pkg/compiler/types/artifact"
	compression "github.com/geaaru/luet/pkg/compiler/types/compression"
	"github.com/geaaru/luet/pkg/config"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type WagonDocument struct {
	Filename        string                     `json:"filename" yaml:"filename"`
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

/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package mask

import (
	cfg "github.com/geaaru/luet/pkg/config"

	gentoo "github.com/geaaru/pkgs-checker/pkg/gentoo"
)

type PackageMaskFile struct {
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Enabled     bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Rules       []string `yaml:"rules" json:"rules"`

	File    string                             `yaml:"-" json:"-"`
	pkgsMap map[string][]*gentoo.GentooPackage `yaml:"-" json:"-"`
}

type PackagesMaskManager struct {
	Config *cfg.LuetConfig

	Files []*PackageMaskFile
}

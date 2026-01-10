/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type ConfigProtectConfFile struct {
	Filename string

	Name        string   `mapstructure:"name" yaml:"name" json:"name"`
	Directories []string `mapstructure:"dirs" yaml:"dirs" json:"dirs"`
}

func NewConfigProtectConfFile(filename string) *ConfigProtectConfFile {
	return &ConfigProtectConfFile{
		Filename:    filename,
		Name:        "",
		Directories: []string{},
	}
}

func (c *ConfigProtectConfFile) String() string {
	return fmt.Sprintf("[%s] filename: %s, dirs: %s", c.Name, c.Filename,
		c.Directories)
}

type ConfigProtect struct {
	AnnotationDir string
	MapProtected  map[string]bool
}

func NewConfigProtect(annotationDir string) *ConfigProtect {
	if len(annotationDir) > 0 && annotationDir[0:1] != "/" {
		annotationDir = "/" + annotationDir
	}
	return &ConfigProtect{
		AnnotationDir: annotationDir,
		MapProtected:  make(map[string]bool, 0),
	}
}

func (c *ConfigProtect) AddAnnotationDir(d string) {
	c.AnnotationDir = d
}

func (c *ConfigProtect) GetAnnotationDir() string {
	return c.AnnotationDir
}

func (c *ConfigProtect) Map(files []string) {
	if AniseCfg.ConfigProtectSkip {
		return
	}

	for _, file := range files {

		if file[0:1] != "/" {
			file = "/" + file
		}

		if len(AniseCfg.GetConfigProtectConfFiles()) > 0 {
			for _, conf := range AniseCfg.GetConfigProtectConfFiles() {
				for _, dir := range conf.Directories {
					// Note file is without / at begin (on unpack)
					if strings.HasPrefix(file, filepath.Clean(dir)) {
						// docker archive modifier works with path without / at begin.
						c.MapProtected[file] = true
						goto nextFile
					}
				}
			}
		}

		if c.AnnotationDir != "" && strings.HasPrefix(file, filepath.Clean(c.AnnotationDir)) {
			c.MapProtected[file] = true
		}
	nextFile:
	}

}

func (c *ConfigProtect) Protected(file string) bool {
	if file[0:1] != "/" {
		file = "/" + file
	}
	_, ans := c.MapProtected[file]
	return ans
}

func (c *ConfigProtect) GetProtectFiles(withSlash bool) []string {
	ans := []string{}

	for key, _ := range c.MapProtected {
		if withSlash {
			ans = append(ans, key)
		} else {
			ans = append(ans, key[1:])
		}
	}
	return ans
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type StonesSearchOpts struct {
	Packages      pkg.DefaultPackages
	Categories    []string
	Labels        []string
	LabelsMatches []string
	Matches       []string
	FilesOwner    []string
	Annotations   []string
	Hidden        bool
	AndCondition  bool
	WithFiles     bool
}

type ArtifactIndex []*artifact.PackageArtifact

type StonesCatalog struct {
	Index ArtifactIndex `json:"index" yaml:"index"`
}

type Stone struct {
	Name        string                 `json:"name" yaml:"name"`
	Category    string                 `json:"category" yaml:"category"`
	Version     string                 `json:"version" yaml:"version"`
	License     string                 `json:"license,omitempty" yaml:"license,omitempty"`
	Repository  string                 `json:"repository" yaml:"repository"`
	Hidden      bool                   `json:"hidden,omitempty" yaml:"hidden,omitempty"`
	Files       []string               `json:"files,omitempty" yaml:"files,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type StonesPack struct {
	Stones []*Stone `json:"stones" yaml:"stones"`
}

type StonesMap struct {
	Stones map[string][]*Stone `json:"stones" yaml:"stones"`
}

type WagonStones struct {
	Catalog *StonesCatalog
}

func (sp *StonesPack) ToMap() *StonesMap {

	ans := &StonesMap{
		Stones: make(map[string][]*Stone, 1),
	}

	for idx, _ := range sp.Stones {
		ans.Add(sp.Stones[idx])
	}

	return ans
}

func (sm *StonesMap) Add(s *Stone) {
	if val, ok := sm.Stones[s.GetName()]; ok {
		sm.Stones[s.GetName()] = append(val, s)
	} else {
		sm.Stones[s.GetName()] = []*Stone{s}
	}
}

func (sm *StonesMap) AddPack(sp *StonesPack) {
	for idx, _ := range sp.Stones {
		sm.Add(sp.Stones[idx])
	}
}

func NewWagonStones() *WagonStones {
	return &WagonStones{
		Catalog: nil,
	}
}

func NewStone(p *artifact.PackageArtifact, repo string, withFiles bool) *Stone {
	ans := &Stone{
		Name:        p.Runtime.Name,
		Category:    p.Runtime.Category,
		Version:     p.Runtime.Version,
		License:     p.Runtime.License,
		Repository:  repo,
		Hidden:      p.Runtime.Hidden,
		Annotations: p.Runtime.Annotations,
		Labels:      p.Runtime.Labels,
	}

	if withFiles {
		ans.Files = p.Files
	}

	return ans
}

func (s *Stone) ToPackage() *pkg.DefaultPackage {
	ans := pkg.NewPackage(s.Name, s.Version, []*pkg.DefaultPackage{}, []*pkg.DefaultPackage{})
	ans.Labels = s.Labels
	ans.Annotations = s.Annotations
	ans.Hidden = s.Hidden
	ans.Category = s.Category
	ans.License = s.License
	ans.Repository = s.Repository

	return ans
}

func (s *Stone) HumanReadableString() string {
	return fmt.Sprintf("%s/%s-%s", s.Category, s.Name, s.Version)
}

func (s *Stone) GetName() string {
	if s.Category != "" && s.Name != "" {
		return fmt.Sprintf("%s/%s", s.Category, s.Name)
	} else if s.Category != "" {
		return s.Category
	} else {
		return s.Name
	}
}

func (s *WagonStones) SearchArtifacts(opts *StonesSearchOpts, repoName string) (*[]*artifact.PackageArtifact, error) {
	ans := []*artifact.PackageArtifact{}

	if s.Catalog == nil {
		return &ans, nil
	}

	if len(opts.LabelsMatches) > 0 && len(opts.Matches) > 0 {
		return nil, errors.New("Searching for both regex and labels regex is not supported.")
	}

	// Create regexes array
	regs := []*regexp.Regexp{}
	lRegs := []*regexp.Regexp{}
	catRegs := []*regexp.Regexp{}

	if len(opts.Matches) > 0 {
		for _, m := range opts.Matches {
			r := regexp.MustCompile(m)
			if r != nil {
				regs = append(regs, r)
			}
		}
	}

	if len(opts.LabelsMatches) > 0 {
		for _, m := range opts.LabelsMatches {
			r := regexp.MustCompile(m)
			if r != nil {
				lRegs = append(lRegs, r)
			}
		}
	}

	if len(opts.Categories) > 0 {
		for _, m := range opts.Categories {
			r := regexp.MustCompile(m)
			if r != nil {
				catRegs = append(catRegs, r)
			}
		}
	}

	for idx, _ := range s.Catalog.Index {
		artifact := s.Catalog.Index[idx]
		if artifact.Runtime == nil {
			//fmt.Println("ARTIFACT ", artifact, repoName)
			Warning(fmt.Sprintf(
				"[%s/%s-%s] Found artifact without runtime pkg. Using compile spec package.",
				artifact.CompileSpec.Package.Category,
				artifact.CompileSpec.Package.Name,
				artifact.CompileSpec.Package.Version,
			))

			artifact.Runtime = artifact.CompileSpec.Package
		}
		if !opts.Hidden && artifact.Runtime.Hidden {
			// Exclude hidden packages
			continue
		}

		match := false

		// For now only match category and name
		if len(opts.Packages) > 0 {
			for idx, _ := range opts.Packages {
				if artifact.Runtime.Category != opts.Packages[idx].GetCategory() {
					continue
				}

				if artifact.Runtime.Name != opts.Packages[idx].GetName() {
					continue
				}

				match = true
				break
			}
		}

		if len(opts.Matches) > 0 {
			pstring := artifact.Runtime.PackageName()

			for ri, _ := range regs {
				if regs[ri].MatchString(pstring) {
					match = true
					break
				}
			}
		}

		if len(opts.LabelsMatches) > 0 {
			if opts.AndCondition {
				match = false
			} else {
				goto matched
			}

			if !match {
				for ri, _ := range lRegs {
					if artifact.Runtime.MatchLabel(lRegs[ri]) {
						match = true
						break
					}
				}
			}
		}

		if len(opts.Labels) > 0 {
			if opts.AndCondition {
				match = false
			} else {
				goto matched
			}

			if !match {
				for _, l := range opts.Labels {
					if artifact.Runtime.HasLabel(l) {
						match = true
						break
					}
				}
			}
		}

		if len(opts.Categories) > 0 {
			if opts.AndCondition {
				match = false
			} else {
				goto matched
			}

			if !match {
				for ri, _ := range catRegs {
					if catRegs[ri].MatchString(artifact.Runtime.Category) {
						match = true
						break
					}
				}
			}
		}

		if len(opts.Annotations) > 0 {
			for _, a := range opts.Annotations {
				if artifact.Runtime.HasAnnotation(a) {
					match = true
					break
				}
			}
		}

		if len(opts.FilesOwner) > 0 {
			if opts.AndCondition {
				match = false
			}

			if len(artifact.Files) > 0 {
				for _, f := range opts.FilesOwner {
					for fidx, _ := range artifact.Files {
						if strings.Index(artifact.Files[fidx], f) >= 0 {
							match = true
							goto matched
						}
					}
				}
			}
		}

	matched:
		if match {
			// Propagate repository information
			if artifact.Runtime != nil {
				artifact.Runtime.Repository = repoName
			} else if artifact.CompileSpec != nil && artifact.CompileSpec.Package != nil {
				artifact.CompileSpec.Package.Repository = repoName
			}

			ans = append(ans, artifact)
		}
	}

	return &ans, nil

}

func (s *WagonStones) Search(opts *StonesSearchOpts, repoName string) (*[]*Stone, error) {
	ans := []*Stone{}

	artifactsMatched, err := s.SearchArtifacts(opts, repoName)
	if err != nil {
		return nil, err
	}

	matches := *artifactsMatched

	if len(matches) > 0 {
		for idx, _ := range matches {
			ans = append(ans, NewStone(matches[idx], repoName, opts.WithFiles))
		}
	}

	return &ans, nil
}

func (s *WagonStones) LoadCatalog(identity *WagonIdentity) (*StonesCatalog, error) {
	ans := &StonesCatalog{}

	repobasedir := filepath.Dir(identity.IdentityFile)

	start := time.Now()

	// TODO: Here we need to handle the new repository style
	//       when ready.
	if _, ok := identity.RepositoryFiles[REPOFILE_META_KEY]; ok {
		metafs := filepath.Join(repobasedir, "metafs")
		metafile := filepath.Join(metafs, REPOSITORY_METAFILE)

		Debug(fmt.Sprintf("[%s] Found metafile %s", identity.Name, metafile))

		/*
			data, err := ioutil.ReadFile(metafile)
			if err != nil {
				return nil, errors.Wrap(err, "Error on reading file "+metafile)
			}

			err = yaml.Unmarshal(data, &ans)
			if err != nil {
				return nil, errors.Wrap(err, "Error on parse file "+metafile)
			}
		*/
		file, err := os.Open(metafile)
		if err != nil {
			return nil, errors.Wrap(err, "Error on reading file "+metafile)
		}
		defer file.Close()

		//decoder := yaml.NewDecoder(file)
		decoder := yaml.NewDecoder(bufio.NewReader(file))
		err = decoder.Decode(&ans)
		if err != nil {
			return nil, errors.Wrap(err, "Error on parse file "+metafile)
		}

	} else {
		return nil, errors.New("No meta field found. Repository is corrupted or to sync.")
	}

	s.Catalog = ans

	Debug(fmt.Sprintf("[%s] metadata loaded in %d µs.", identity.Name,
		time.Now().Sub(start).Nanoseconds()/1e3))

	return ans, nil
}

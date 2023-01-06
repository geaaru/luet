/*
Copyright © 2022 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"path/filepath"

	"github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	art "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	"github.com/pkg/errors"
)

type Searcher interface {
	SearchArtifacts(searchOpts *StonesSearchOpts) (*[]*art.PackageArtifact, error)
	SearchStones(searchOpts *StonesSearchOpts) (*[]*Stone, error)
	SearchInstalled(searchOpts *StonesSearchOpts) (*[]*Stone, error)
}

type SearcherSimple struct {
	Config   *config.LuetConfig
	Database pkg.PackageDatabase
}

func NewSearcherSimple(cfg *config.LuetConfig) *SearcherSimple {
	return &SearcherSimple{
		Config:   cfg,
		Database: nil,
	}
}

func (s *SearcherSimple) Setup() {
	if s.Database == nil {
		s.Database = s.Config.GetSystemDB()
	}
}

func (s *SearcherSimple) Close() {
	if s.Database != nil {
		s.Database.Close()
	}
}

func (s *SearcherSimple) searchOnRepoRoutine(
	repo *config.LuetRepository,
	searchOpts *StonesSearchOpts,
	channel chan ChannelSearchRes,
	artifactsRes bool) {

	repobasedir := s.Config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
	r := NewWagonRepository(repo)
	err := r.ReadWagonIdentify(repobasedir)
	if err != nil {

		Warning("Error on read repository identity file: " + err.Error())
		if artifactsRes {
			ansArts := []*art.PackageArtifact{}
			channel <- ChannelSearchRes{nil, &ansArts, nil}
		} else {
			ansStones := []*Stone{}
			channel <- ChannelSearchRes{&ansStones, nil, nil}
		}

	} else {
		if artifactsRes {
			artifacts, err := r.SearchArtifacts(searchOpts)
			if err != nil {
				Warning("Error on read repository catalog for repo : " + r.Identity.Name)
				channel <- ChannelSearchRes{nil, nil, err}
				return
			}

			channel <- ChannelSearchRes{nil, artifacts, nil}
		} else {
			var pkgs *[]*Stone
			var err error

			pkgs, err = r.SearchStones(searchOpts)
			if err != nil {
				Warning("Error on read repository catalog for repo : " + r.Identity.Name)
				channel <- ChannelSearchRes{nil, nil, err}
				return
			}

			channel <- ChannelSearchRes{pkgs, nil, nil}
		}

		r.ClearCatalog()
	}
}

func (s *SearcherSimple) SearchArtifactsOnRepo(name string, searchOpts *StonesSearchOpts) (*[]*art.PackageArtifact, error) {

	repo, err := s.Config.GetSystemRepository(name)
	if err != nil {
		return nil, err
	}

	ans := []*art.PackageArtifact{}
	repobasedir := s.Config.GetSystem().GetRepoDatabaseDirPath(repo.Name)
	r := NewWagonRepository(repo)
	err = r.ReadWagonIdentify(repobasedir)
	if err != nil {
		Warning("Error on read repository identity file: " + err.Error())
	} else {
		artifacts, err := r.SearchArtifacts(searchOpts)
		if err != nil {
			Warning("Error on read repository catalog for repo : " + r.Identity.Name)
			return &ans, err
		}

		ans = *artifacts

		// NOTE: Check if this is correct when this method is used
		//       outside the simple search scope.
		r.ClearCatalog()
	}

	return &ans, nil
}

func (s *SearcherSimple) SearchArtifacts(opts *StonesSearchOpts) (*[]*art.PackageArtifact, error) {

	res := []*art.PackageArtifact{}
	var ch chan ChannelSearchRes = make(
		chan ChannelSearchRes,
		s.Config.GetGeneral().Concurrency,
	)

	for idx, _ := range s.Config.SystemRepositories {
		repo := s.Config.SystemRepositories[idx]
		if !repo.Enable {
			continue
		}

		if repo.Cached {
			go s.searchOnRepoRoutine(&repo, opts, ch, true)
		} else {
			return &res, errors.New("Only cached repositories are supported.")
		}

	}

	var err error = nil

	for idx, _ := range s.Config.SystemRepositories {
		repo := s.Config.SystemRepositories[idx]
		if !repo.Enable {
			continue
		}

		if repo.Cached {
			resp := <-ch
			if resp.Error == nil {
				res = append(res, *resp.Artifacts...)
			} else {
				err = resp.Error
			}
		}
	}

	return &res, err
}

func (s *SearcherSimple) SearchInstalled(opts *StonesSearchOpts) (*[]*Stone, error) {
	s.Setup()

	wagonStones := NewWagonStones()
	wagonStones.Catalog = &StonesCatalog{}

	pkgs := s.Database.World()
	for idx, _ := range pkgs {
		p := pkgs[idx].(*pkg.DefaultPackage)
		artifact := art.NewPackageArtifact(p.GetPath())
		artifact.Runtime = p
		if opts.WithFiles {
			f, _ := s.Database.GetPackageFiles(pkgs[idx])

			if opts.WithRootfsPrefix {
				artifact.Files = []string{}
				for _, ff := range f {
					artifact.Files = append(artifact.Files,
						filepath.Join(s.Config.GetSystem().Rootfs, ff),
					)
				}
			} else {
				artifact.Files = f
			}
		}
		wagonStones.Catalog.Index = append(wagonStones.Catalog.Index, artifact)
	}

	return wagonStones.SearchFromCatalog(opts, "system")
}

func (s *SearcherSimple) SearchStones(opts *StonesSearchOpts) (*[]*Stone, error) {
	res := []*Stone{}
	var ch chan ChannelSearchRes = make(
		chan ChannelSearchRes,
		s.Config.GetGeneral().Concurrency,
	)

	for idx, _ := range s.Config.SystemRepositories {
		repo := s.Config.SystemRepositories[idx]
		if !repo.Enable {
			continue
		}

		if repo.Cached {
			go s.searchOnRepoRoutine(&repo, opts, ch, false)
		} else {
			return &res, errors.New("Only cached repositories are supported.")
		}

	}

	var err error = nil

	for idx, _ := range s.Config.SystemRepositories {
		repo := s.Config.SystemRepositories[idx]
		if !repo.Enable {
			continue
		}

		if repo.Cached {
			resp := <-ch
			if resp.Error == nil {
				res = append(res, *resp.Stones...)
			} else {
				err = resp.Error
			}
		}
	}

	return &res, err
}

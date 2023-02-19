/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package solver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/geaaru/luet/pkg/helpers"
	. "github.com/geaaru/luet/pkg/logger"
	pkg "github.com/geaaru/luet/pkg/package"
	wagon "github.com/geaaru/luet/pkg/v2/repository"
	"golang.org/x/sync/semaphore"
)

func (s *Solver) checkOrphanPackage(
	p *pkg.DefaultPackage, channel chan helpers.ChannelError,
	sem *semaphore.Weighted, waitGroup *sync.WaitGroup,
	ctx *context.Context) {

	defer waitGroup.Done()
	err := sem.Acquire(*ctx, 1)
	if err != nil {
		Error("Error on acquire semaphore: " + err.Error())
		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}
	defer sem.Release(1)

	start := time.Now()

	searchOpts := &wagon.StonesSearchOpts{
		Packages:         []*pkg.DefaultPackage{p},
		Categories:       []string{},
		Labels:           []string{},
		LabelsMatches:    []string{},
		Matches:          []string{},
		FilesOwner:       []string{},
		Annotations:      []string{},
		Hidden:           true,
		AndCondition:     false,
		WithFiles:        false,
		WithRootfsPrefix: false,
		Full:             false,
		OnlyPackages:     true,
		IgnoreMasks:      s.Opts.IgnoreMasks,
	}

	reposArtifacts, err := s.Searcher.SearchArtifacts(searchOpts)
	if err != nil {
		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}

	if len(*reposArtifacts) == 0 {
		// The package is not more available between the active
		// repositories
		Debug(fmt.Sprintf("[%s] No packages found on repository.", p.PackageName()))

		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}

	Debug(fmt.Sprintf(":brain: Analysis %s done in %d µs.",
		p.PackageName(),
		time.Now().Sub(start).Nanoseconds()/1e3))
	if err != nil {
		channel <- helpers.ChannelError{
			Error:   err,
			Closure: p,
		}
		return
	}

	channel <- helpers.ChannelError{
		Error:   nil,
		Closure: nil,
	}
	return
}

func (s *Solver) Orphans() (*[]*pkg.DefaultPackage, error) {
	ans := []*pkg.DefaultPackage{}

	// TODO: Use a different solution with less memory usage
	systemPkgs := s.Database.World()

	s.prepareConflictsAndSystemMap(&systemPkgs, false)

	// Prepare the searcher
	err := s.prepareSearcher()
	if err != nil {
		return &ans, err
	}

	waitGroup := &sync.WaitGroup{}
	sem := semaphore.NewWeighted(int64(
		s.Config.GetGeneral().Concurrency))
	ctx := context.TODO()

	defer waitGroup.Wait()

	var ch chan helpers.ChannelError = make(
		chan helpers.ChannelError,
		s.Config.GetGeneral().Concurrency,
	)

	for _, p := range s.systemMap.Packages {
		waitGroup.Add(1)
		go s.checkOrphanPackage(p[0], ch, sem, waitGroup, &ctx)
	}

	fail := false
	nPkgs := len(s.systemMap.Packages)
	if nPkgs > 0 {
		for i := 0; i < nPkgs; i++ {
			resp := <-ch
			if resp.Error != nil {
				Error(fmt.Sprintf("[%s] Error: %s",
					(resp.Closure.(*pkg.DefaultPackage)).PackageName(),
					resp.Error.Error()))
				fail = true
			} else if resp.Closure != nil {
				ans = append(ans, resp.Closure.(*pkg.DefaultPackage))
			}
		}
	}

	if fail {
		return nil, errors.New("Orphans research interrupted for errors.")
	}

	return &ans, nil
}

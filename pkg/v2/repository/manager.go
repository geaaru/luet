/*
Copyright © 2019-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"context"
	"sync"

	"github.com/geaaru/luet/pkg/config"
	cfg "github.com/geaaru/luet/pkg/config"
	. "github.com/geaaru/luet/pkg/logger"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

type WagonsRails struct {
	Config *cfg.LuetConfig
}

type SyncOpts struct {
	IgnoreErrors bool
	Force        bool
}

type ChannelRepoOpRes struct {
	Error error
	Repo  *config.LuetRepository
}

func NewWagonsRails(c *cfg.LuetConfig) *WagonsRails {
	return &WagonsRails{
		Config: c,
	}
}

func (w *WagonsRails) processRepository(
	repo *cfg.LuetRepository,
	channel chan ChannelRepoOpRes,
	force bool, sem *semaphore.Weighted,
	waitGroup *sync.WaitGroup, ctx *context.Context) {

	repobasedir := w.Config.GetSystem().GetRepoDatabaseDirPath(repo.Name)

	defer waitGroup.Done()

	err := sem.Acquire(*ctx, 1)
	if err != nil {
		return
	}
	defer sem.Release(1)

	r := NewWagonRepository(repo)
	if r.HasLocalWagonIdentity(repobasedir) {
		err = r.ReadWagonIdentify(repobasedir)
		if err != nil && (!force) {
			channel <- ChannelRepoOpRes{err, repo}
			return
		}
	}

	err = r.Sync(force)
	r.ClearCatalog()
	r = nil

	if err != nil {
		channel <- ChannelRepoOpRes{err, repo}
	} else {
		channel <- ChannelRepoOpRes{nil, repo}
	}
	return
}

func (w *WagonsRails) SyncRepos(repos []string, opts *SyncOpts) error {
	var ch chan ChannelRepoOpRes = make(
		chan ChannelRepoOpRes,
		w.Config.GetGeneral().Concurrency,
	)
	nOps := 0

	waitGroup := &sync.WaitGroup{}
	sem := semaphore.NewWeighted(int64(w.Config.GetGeneral().Concurrency))
	ctx := context.TODO()

	if len(repos) > 0 {
		for _, rname := range repos {
			repo, err := w.Config.GetSystemRepository(rname)
			if err != nil && !opts.IgnoreErrors {
				return err
			} else if err != nil {
				continue
			}
			waitGroup.Add(1)

			go w.processRepository(repo, ch, opts.Force, sem, waitGroup, &ctx)
			nOps++
		}
	} else {
		for idx, repo := range w.Config.SystemRepositories {
			if repo.Enable {
				waitGroup.Add(1)
				go w.processRepository(
					&w.Config.SystemRepositories[idx], ch, opts.Force,
					sem, waitGroup, &ctx,
				)
				nOps++
			}
		}
	}

	if nOps > 0 {
		withErr := false
		for i := 0; i < nOps; i++ {
			resp := <-ch
			if resp.Error != nil && !opts.IgnoreErrors {
				withErr = true
				Error("Error on update repository " + resp.Repo.Name + ": " + resp.Error.Error())
			}
		}

		waitGroup.Wait()

		if withErr {
			return errors.New("Not all repositories are been synced.")
		}

	} else {
		InfoC(":flag_white:No repositories candidates found.")
	}

	return nil
}

/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package repository

import (
	"path/filepath"

	"github.com/geaaru/luet/pkg/config"
)

const (
	REPOSITORY_METAFILE = "repository.meta.yaml"
	REPOSITORY_SPECFILE = "repository.yaml"
)

type WagonRepository struct {
	Identity *WagonIdentity
	Stones   *WagonStones
}

func NewWagonRepository(l *config.LuetRepository) *WagonRepository {
	return &WagonRepository{
		Identity: NewWagonIdentify(l),
		Stones:   NewWagonStones(),
	}
}

func (w *WagonRepository) SearchStones(opts *StonesSearchOpts) (*[]*Stone, error) {

	// Load catalog if not loaded yet
	if w.Stones.Catalog == nil {
		_, err := w.Stones.LoadCatalog(w.Identity)
		if err != nil {
			return nil, err
		}
	}

	return w.Stones.Search(opts, w.Identity.Name)
}

func (w *WagonRepository) ReadWagonIdentify(wdir string) error {
	file := filepath.Join(wdir, REPOSITORY_SPECFILE)

	return w.Identity.Load(file)
}

func (w *WagonRepository) GetRevision() int {
	return w.Identity.LuetRepository.Revision
}
func (w *WagonRepository) GetLastUpdate() string {
	return w.Identity.LuetRepository.LastUpdate
}
func (w *WagonRepository) SetLastUpdate(u string) {
	w.Identity.LuetRepository.LastUpdate = u
}
func (w *WagonRepository) IncrementRevision() {
	w.Identity.LuetRepository.Revision++
}

func (w *WagonRepository) ClearCatalog() {
	w.Stones.Catalog = nil
}

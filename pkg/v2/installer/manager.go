/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	cfg "github.com/geaaru/luet/pkg/config"
	artifact "github.com/geaaru/luet/pkg/v2/compiler/types/artifact"
	repos "github.com/geaaru/luet/pkg/v2/repository"

	"github.com/pkg/errors"
)

type ArtifatctsManager struct {
	Config *cfg.LuetConfig
}

func NewArtifactsManager(config *cfg.LuetConfig) *ArtifatctsManager {
	return &ArtifatctsManager{
		Config: config,
	}
}

func (m *ArtifatctsManager) DownloadPackage(p *artifact.PackageArtifact, r *repos.WagonRepository) error {

	c := r.Client()
	if c == nil {
		return errors.New("No client could be generated from repository")
	}

	err := c.DownloadArtifact(p)
	if err != nil {
		return errors.Wrap(err, "Error on download artifact")
	}

	err = p.Verify()
	if err != nil {
		return errors.Wrap(err,
			"Artifact integrity check failure for file "+p.CachePath)
	}

	return nil
}

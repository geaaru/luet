/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

import (
	cfg "github.com/geaaru/luet/pkg/config"
)

type ArtifatctsManager struct {
	Config *cfg.LuetConfig
}

func NewArtifactsManager(config *cfg.LuetConfig) *ArtifatctsManager {
	return &ArtifatctsManager{
		Config: config,
	}
}

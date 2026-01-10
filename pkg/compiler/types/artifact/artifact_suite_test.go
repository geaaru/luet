/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package artifact_test

import (
	"testing"

	. "github.com/macaroni-os/anise/cmd"
	config "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifact(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.LuetCfg)
	RunSpecs(t, "Artifact Suite")
}

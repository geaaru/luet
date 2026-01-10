/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package client_test

import (
	"testing"

	. "github.com/macaroni-os/anise/cmd"
	config "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.AniseCfg)
	// Set temporary directory for rootfs
	config.AniseCfg.GetSystem().Rootfs = "/tmp/anise-root"
	// Force dynamic path for packages cache
	config.AniseCfg.GetSystem().PkgsCachePath = ""
	RunSpecs(t, "Client Suite")
}

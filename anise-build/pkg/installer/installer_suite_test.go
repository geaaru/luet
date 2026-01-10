/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package installer_test

import (
	"testing"

	. "github.com/macaroni-os/anise/cmd"
	config "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstaller(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.LuetCfg)
	// Set temporary directory for rootfs
	config.LuetCfg.GetSystem().Rootfs = "/tmp/luet-root"
	// Force dynamic path for packages cache
	config.LuetCfg.GetSystem().PkgsCachePath = ""
	RunSpecs(t, "Installer Suite")
}

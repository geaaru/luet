/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package version_test

import (
	"testing"

	. "github.com/macaroni-os/anise/cmd"
	config "github.com/macaroni-os/anise/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVersioner(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.AniseCfg)
	RunSpecs(t, "Versioner Suite")
}

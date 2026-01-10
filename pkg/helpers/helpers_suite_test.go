/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package helpers_test

import (
	"testing"

	. "github.com/macaroni-os/anise/cmd"
	config "github.com/macaroni-os/anise/pkg/config"

	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSolver(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.AniseCfg)

	cfg := tarf_specs.NewConfig(config.AniseCfg.Viper)
	cfg.GetGeneral().Debug = config.AniseCfg.GetGeneral().Debug
	cfg.GetLogging().Level = config.AniseCfg.GetLogging().Level

	tf := tarf.NewTarFormers(cfg)
	tarf.SetDefaultTarFormers(tf)

	RunSpecs(t, "Helpers Suite")
}

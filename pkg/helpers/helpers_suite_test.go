// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package helpers_test

import (
	"testing"

	. "github.com/mudler/luet/cmd"
	config "github.com/mudler/luet/pkg/config"

	tarf "github.com/geaaru/tar-formers/pkg/executor"
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSolver(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.LuetCfg)

	cfg := tarf_specs.NewConfig(config.LuetCfg.Viper)
	cfg.GetGeneral().Debug = config.LuetCfg.GetGeneral().Debug
	cfg.GetLogging().Level = config.LuetCfg.GetLogging().Level

	tf := tarf.NewTarFormers(cfg)
	tarf.SetDefaultTarFormers(tf)

	RunSpecs(t, "Helpers Suite")
}

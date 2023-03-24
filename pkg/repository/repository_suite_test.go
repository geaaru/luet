/*
Copyright © 2021-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package repository_test

import (
	"testing"

	. "github.com/geaaru/luet/cmd"
	config "github.com/geaaru/luet/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSolver(t *testing.T) {
	RegisterFailHandler(Fail)
	LoadConfig(config.LuetCfg)
	RunSpecs(t, "Repository Suite")
}

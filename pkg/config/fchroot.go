/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package config

import "fmt"

type FchrootOpts struct {
	Verbose bool   `yaml:"verbose,omitempty" json:"verbose,omitempty" mapstructure:"verbose"`
	Debug   bool   `yaml:"debug,omitempty" json:"debug,omitempty" mapstructure:"debug"`
	Cpu     string `yaml:"cpu,omitempty" json:"cpu,omitempty" mapstructure:"cpu"`
	NoBind  bool   `yaml:"nobind,omitempty" json:"nobind,omitempty" mapstructure:"nobind"`
}

func (o *FchrootOpts) GetFlags(binds []string) []string {
	ans := []string{
		// We want always this because the env is passed
		// by anise with his environment variables.
		"--preserve-env",
	}

	if o.Verbose {
		ans = append(ans, "-v")
	}
	if o.Debug {
		ans = append(ans, "--debug")
	}
	if o.Cpu != "" {
		ans = append(ans, "--cpu="+o.Cpu)
	}
	if o.NoBind {
		ans = append(ans, "--nobind")
	}

	for _, v := range binds {
		ans = append(ans,
			fmt.Sprintf("--bind=%s", v))
	}

	return ans
}

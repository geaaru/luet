/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package artifact

import (
	"fmt"

	. "github.com/macaroni-os/anise/pkg/config"
	"github.com/macaroni-os/anise/pkg/helpers"
	. "github.com/macaroni-os/anise/pkg/logger"
	pkg "github.com/macaroni-os/anise/pkg/package"

	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
)

func (a *PackageArtifact) GetTarFormersSpec(enableSubsets bool) *tarf_specs.SpecFile {
	spec := tarf_specs.NewSpecFile()
	spec.SameOwner = AniseCfg.GetGeneral().SameOwner
	spec.EnableMutex = AniseCfg.GetTarFlows().Mutex4Dirs
	spec.MaxOpenFiles = AniseCfg.GetTarFlows().MaxOpenFiles
	spec.BufferSize = AniseCfg.GetTarFlows().CopyBufferSize
	spec.OverwritePerms = AniseCfg.GetGeneral().OverwriteDirPerms
	spec.IgnoreRegexes = []string{
		// prevent 'operation not permitted'
		//"^/dev",
	}
	spec.IgnoreFiles = []string{}

	if enableSubsets {
		def := a.GetSubsets()

		for k, v := range def.Definitions {
			if !helpers.ContainsElem(&AniseCfg.Subsets.Enabled, k) {
				// POST: the selected subset is not enabled.
				//       I add the rules as IgnoreRegexes.

				for _, r := range v.Rules {
					spec.IgnoreRegexes = append(spec.IgnoreRegexes, r)
				}

				Debug(fmt.Sprintf("[%s] Adding ignore regexes %s",
					a.Runtime.HumanReadableString(), v.Rules))

			}
		}
	}

	return spec
}

func (a *PackageArtifact) GetSubsets() *AniseSubsetsDefinition {
	ans := &AniseSubsetsDefinition{
		Definitions: make(map[string]*AniseSubsetDefinition, 0),
	}

	if a.Runtime != nil {
		// Get global/user category subsets defined
		catSubsets := AniseCfg.SubsetsCatDefMap[a.Runtime.GetCategory()]

		// Get global/user package subsets defined
		pkgSubsets := AniseCfg.SubsetsPkgsDefMap[a.Runtime.PackageName()]

		// Check if there is subsets definition on annotations
		if a.Runtime.HasAnnotation(string(pkg.SubsetsAnnotation)) {
			ans = a.unmarshalSubsets()
		}

		// Respect this order on override the subsets:
		// If there are subsets on package annotation I set the
		// initial definitions.
		// If there is a pkg subsets defined locally i use them
		// to add new subsets or override existing with the same keys.
		// If there isn't a pkg subsets and there are subsets
		// defined for the category i override the subsets at the
		// same way.

		if pkgSubsets != nil {
			for k, v := range pkgSubsets.Definitions {
				ans.Definitions[k] = v
			}
		} else if catSubsets != nil {
			for k, v := range catSubsets.Definitions {
				ans.Definitions[k] = v
			}
		}
	}

	return ans
}

func (a *PackageArtifact) unmarshalSubsets() *AniseSubsetsDefinition {

	ans := &AniseSubsetsDefinition{
		Definitions: make(map[string]*AniseSubsetDefinition, 0),
	}

	subsets := a.Runtime.GetAnnotationByKey(
		string(pkg.SubsetsAnnotation),
	)

	obj, ok := subsets.(map[interface{}]interface{})

	if !ok {
		Warning(fmt.Sprintf("[%s] Wrong format on %s annotation.",
			a.Runtime.HumanReadableString(), pkg.SubsetsAnnotation,
		))
		return ans
	}

	for k, v := range obj {

		krules, kok := k.(string)
		if !kok {
			Warning(fmt.Sprintf("[%s] Invalid key on subset %s.",
				a.Runtime.HumanReadableString(), k,
			))
			continue
		}

		if krules != "rules" {
			continue
		}

		mrules, ok := v.(map[interface{}]interface{})
		if !ok {
			Warning(fmt.Sprintf("[%s] Wrong rules on subset %s.",
				a.Runtime.HumanReadableString(), k,
			))
			continue
		}

		for mk, vrules := range mrules {

			kk := mk.(string)
			rules := []string{}
			irules, ok := vrules.([]interface{})
			if !ok {
				Warning(fmt.Sprintf("[%s] For subset %s value is not an array.",
					a.Runtime.HumanReadableString(), kk,
				))
				continue
			}

			for _, r := range irules {
				rules = append(rules, r.(string))
			}

			ans.Definitions[kk] = &AniseSubsetDefinition{
				Name:  kk,
				Rules: rules,
			}
		}

	}

	return ans
}

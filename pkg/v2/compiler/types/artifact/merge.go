/*
Copyright © 2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package artifact

import (
	pkg "github.com/geaaru/luet/pkg/package"
)

func (a *PackageArtifact) MergeDefinition(dp *pkg.DefaultPackage) {
	if a.Runtime != nil {
		if len(dp.Provides) != len(a.Runtime.Provides) || len(dp.Provides) > 0 {
			a.Runtime.Provides = dp.Provides
			a.CompileSpec.Package.Provides = dp.Provides
		}

		if len(dp.PackageRequires) > 0 || len(dp.PackageRequires) != len(a.Runtime.PackageRequires) {
			a.Runtime.PackageRequires = dp.PackageRequires
			a.CompileSpec.Package.PackageRequires = dp.PackageRequires
		}

		// Update annotations
		if len(dp.Annotations) > 0 || len(dp.Annotations) != len(a.Runtime.Annotations) {
			a.Runtime.Annotations = dp.Annotations
			a.CompileSpec.Package.Annotations = dp.Annotations
		}

	} else if a.CompileSpec != nil && a.CompileSpec.Package != nil {

		if len(dp.Provides) != len(a.CompileSpec.Package.Provides) || len(dp.Provides) > 0 {
			a.CompileSpec.Package.Provides = dp.Provides
		}

		if len(dp.PackageRequires) > 0 || len(dp.PackageRequires) != len(a.CompileSpec.Package.PackageRequires) {
			a.CompileSpec.Package.PackageRequires = dp.PackageRequires
		}

		// Update annotations
		if dp.Annotations != nil && a.CompileSpec.Package.Annotations != nil {
			if len(dp.Annotations) > 0 || len(dp.Annotations) != len(a.CompileSpec.Package.Annotations) {
				a.CompileSpec.Package.Annotations = dp.Annotations
			}
		}
	}
}

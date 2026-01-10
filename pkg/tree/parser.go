/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package tree

import pkg "github.com/macaroni-os/anise/pkg/package"

// parses ebuilds (?) and generates data which is readable by the builder
type Parser interface {
	Generate(string) (pkg.PackageDatabase, error) // Generate scannable luet tree (by builder)
}

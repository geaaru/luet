/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package tree

import (
	pkg "github.com/macaroni-os/anise/pkg/package"
)

// reads a anise tree and generates the package lists
type Builder interface {
	Save(string) error // A tree might be saved to a folder structure (human editable)
	Load(string) error // A tree might be loaded from a db (e.g. bolt) and written to folder
	GetDatabase() pkg.PackageDatabase
	WithDatabase(d pkg.PackageDatabase)

	GetSourcePath() []string
}

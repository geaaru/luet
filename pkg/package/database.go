/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package pkg

// Database is a merely simple in-memory db.
// FIXME: Use a proper structure or delegate to third-party
type PackageDatabase interface {
	PackageSet

	Get(s string) (string, error)
	Set(k, v string) error

	Create(string, []byte) (string, error)
	Retrieve(ID string) ([]byte, error)

	Close() error
}

type PackageSet interface {
	Clone(PackageDatabase) error
	Copy() (PackageDatabase, error)

	GetRevdeps(p Package) (Packages, error)
	GetPackages() []string //Ids
	CreatePackage(pkg Package) (string, error)
	GetPackage(ID string) (Package, error)
	Clean() error
	FindPackage(Package) (Package, error)
	FindPackages(p Package) (Packages, error)
	UpdatePackage(p Package) error
	GetAllPackages(packages chan Package) error
	RemovePackage(Package) error

	GetPackageFiles(Package) ([]string, error)
	SetPackageFiles(*PackageFile) error
	RemovePackageFiles(Package) error
	FindPackageVersions(p Package) (Packages, error)
	World() Packages

	// Finalizers
	GetPackageFinalizer(Package) (*PackageFinalizer, error)
	SetPackageFinalizer(*PackageFinalizer) error
	RemovePackageFinalizer(Package) error

	FindPackageCandidate(p Package) (Package, error)
	FindPackageLabel(labelKey string) (Packages, error)
	FindPackageLabelMatch(pattern string) (Packages, error)
	FindPackageMatch(pattern string) (Packages, error)
	FindPackageByFile(pattern string) (Packages, error)

	RebuildIndexes() error
}

type PackageFile struct {
	ID                 int    `storm:"id,increment"` // primary key with auto increment
	PackageFingerprint string `storm:"unique"`
	Files              []string
}

type PackageFinalizer struct {
	ID                 int      `storm:"id,increment"` // primary key with auto increment
	PackageFingerprint string   `storm:"unique"`
	Shell              []string `json:"shell,omitempty" yaml:"shell,omitempty"`
	Install            []string `json:"install,omitempty" yaml:"install,omitempty"`
	Uninstall          []string `json:"uninstall,omitempty" yaml:"uninstall,omitempty"`
}

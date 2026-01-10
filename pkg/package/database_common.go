/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package pkg

import (
	"regexp"

	"github.com/pkg/errors"
)

func clone(src, dst PackageDatabase) error {
	for _, i := range src.World() {
		_, err := dst.CreatePackage(i)
		if err != nil {
			return errors.Wrap(err, "Failed create package "+i.HumanReadableString())
		}
	}
	return nil
}

func copy(src PackageDatabase) (PackageDatabase, error) {
	dst := NewInMemoryDatabase(false)

	if err := clone(src, dst); err != nil {
		return dst, errors.Wrap(err, "Failed create temporary in-memory db")
	}

	return dst, nil
}

func findPackageByFile(db PackageDatabase, pattern string) (Packages, error) {

	var ans []Package

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid regex "+pattern+"!")
	}

PACKAGE:
	for _, pack := range db.World() {
		files, err := db.GetPackageFiles(pack)
		if err == nil {
			for _, f := range files {
				if re.MatchString(f) {
					ans = append(ans, pack)
					continue PACKAGE
				}
			}
		}
	}

	return Packages(ans), nil

}

/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package artifact

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/pkg/errors"
)

type HashImplementation string

const (
	SHA256 HashImplementation = "sha256"
)

type Checksums map[string]string

type HashOptions struct {
	Hasher hash.Hash
	Type   HashImplementation
}

// Generate generates all Checksums supported for the artifact
func (c *Checksums) Generate(a *PackageArtifact) error {
	return c.generateSHA256(a)
}

func (c Checksums) Compare(d Checksums) error {
	for t, sum := range d {
		if v, ok := c[t]; ok && v != sum {
			return errors.New("Checksum mismsatch")
		}
	}
	return nil
}

func (c *Checksums) generateSHA256(a *PackageArtifact) error {
	return c.generateSum(a, HashOptions{Hasher: sha256.New(), Type: SHA256})
}

func (c *Checksums) generateSum(a *PackageArtifact, opts HashOptions) error {

	f, err := os.Open(a.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(opts.Hasher, f); err != nil {
		return err
	}

	sum := fmt.Sprintf("%x", opts.Hasher.Sum(nil))

	(*c)[string(opts.Type)] = sum
	return nil
}

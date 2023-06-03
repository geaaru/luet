/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package compression

type Implementation string

const (
	None      Implementation = "none" // e.g. tar for standard packages
	GZip      Implementation = "gzip"
	Zstandard Implementation = "zstd"
)

func NewCompression(s string) Implementation {
	switch s {
	case "gzip":
		return GZip
	case "zstd":
		return Zstandard
	default:
		return None
	}
}

func (c Implementation) Ext() string {
	switch c {
	case GZip:
		return ".gz"
	case Zstandard:
		return ".zst"
	default:
		return ""
	}
}

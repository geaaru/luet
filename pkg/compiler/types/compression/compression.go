/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package compression

type Implementation string

const (
	None      Implementation = "none" // e.g. tar for standard packages
	GZip      Implementation = "gzip"
	Zstandard Implementation = "zstd"
)

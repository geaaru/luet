/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package backend

import (
	"bytes"

	. "github.com/macaroni-os/anise/pkg/logger"
)

type BackendWriter struct {
	BufferedOutput bool
	Buffer         *bytes.Buffer
}

func NewBackendWriter(buffered bool) *BackendWriter {
	return &BackendWriter{
		BufferedOutput: buffered,
		Buffer:         &bytes.Buffer{},
	}
}

func (b *BackendWriter) Write(p []byte) (int, error) {
	if b.BufferedOutput {
		return b.Buffer.Write(p)
	}

	Msg("info", false, false, (string(p)))

	return len(p), nil
}

func (b *BackendWriter) Close() error              { return nil }
func (b *BackendWriter) GetCombinedOutput() string { return b.Buffer.String() }

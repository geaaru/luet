/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package version

// Versioner is responsible of sanitizing versions,
// validating them and ordering by precedence
type Versioner interface {
	Sanitize(string) string
	Validate(string) error
	Sort([]string) []string

	ValidateSelector(version string, selector string) bool
}

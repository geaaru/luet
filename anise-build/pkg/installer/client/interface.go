/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package client

type RepoData struct {
	Urls           []string
	Authentication map[string]string
	Verify         bool
}

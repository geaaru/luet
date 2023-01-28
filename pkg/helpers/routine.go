/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

type ChannelError struct {
	Error   error
	Closure interface{}
}

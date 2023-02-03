/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package tools

func Ternary[T any](condition bool, If, Else T) T {
	if condition {
		return If
	}
	return Else
}

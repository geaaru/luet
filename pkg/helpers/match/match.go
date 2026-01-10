/*
Copyright © 2019-2026 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/

package match

import (
	"reflect"
	"regexp"
)

func ReverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func MapIMatchRegex(m *map[string]interface{}, r *regexp.Regexp) bool {
	ans := false

	if m != nil {
		for k, _ := range *m {
			if r.MatchString(k) {
				ans = true
				break
			}
		}
	}

	return ans
}

func MapMatchRegex(m *map[string]string, r *regexp.Regexp) bool {
	ans := false

	if m != nil {
		for k, v := range *m {
			if r.MatchString(k + "=" + v) {
				ans = true
				break
			}
		}
	}

	return ans
}

func MapIHasKey(m *map[string]interface{}, label string) bool {
	ans := false
	if m != nil {
		for k, _ := range *m {
			if k == label {
				ans = true
				break
			}
		}
	}
	return ans
}

func MapHasKey(m *map[string]string, label string) bool {
	ans := false
	if m != nil {
		for k, _ := range *m {
			if k == label {
				ans = true
				break
			}
		}
	}
	return ans
}

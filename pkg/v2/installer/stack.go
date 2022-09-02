/*
	Copyright © 2022 Macaroni OS Linux
	See AUTHORS and LICENSE for the license details and contributors.
*/
package installer

type Stack struct {
	Packages []string
}

func NewStack() *Stack {
	return &Stack{
		Packages: []string{},
	}
}

func (s *Stack) AddPackage(pkg string) {
	s.Packages = append(s.Packages, pkg)
}

func (s *Stack) HasPackage(pkg string) bool {
	ans := false
	for _, p := range s.Packages {
		if p == pkg {
			ans = true
			break
		}
	}
	return ans
}

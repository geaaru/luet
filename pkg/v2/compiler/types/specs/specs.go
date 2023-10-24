/*
Copyright © 2022-2023 Macaroni OS Linux
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

func NewCompilationspecs(s ...*CompilationSpec) *Compilationspecs {
	all := Compilationspecs{}

	for _, spec := range s {
		all.Add(spec)
	}
	return &all
}

func (specs Compilationspecs) Len() int {
	return len(specs)
}

func (specs *Compilationspecs) Remove(s *Compilationspecs) *Compilationspecs {
	newSpecs := Compilationspecs{}
SPECS:
	for _, spec := range specs.All() {
		for _, target := range s.All() {
			if target.GetPackage().Matches(spec.GetPackage()) {
				continue SPECS
			}
		}
		newSpecs.Add(spec)
	}
	return &newSpecs
}

func (specs *Compilationspecs) Add(s *CompilationSpec) {
	*specs = append(*specs, *s)
}

func (specs *Compilationspecs) All() []*CompilationSpec {
	var cspecs []*CompilationSpec
	for i, _ := range *specs {
		f := (*specs)[i]
		cspecs = append(cspecs, &f)
	}

	return cspecs
}

func (specs *Compilationspecs) Unique() *Compilationspecs {
	newSpecs := Compilationspecs{}
	seen := map[string]bool{}

	for i, _ := range *specs {
		j := (*specs)[i]
		_, ok := seen[j.GetPackage().GetFingerPrint()]
		if !ok {
			seen[j.GetPackage().GetFingerPrint()] = true
			newSpecs = append(newSpecs, j)
		}
	}
	return &newSpecs
}

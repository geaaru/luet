package helpers

func Contains(s []string, e string) bool {
	return ContainsElem(&s, e)
}

func ContainsElem(a *[]string, e string) bool {
	s := *a
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

package fauna

func arrayContains[T comparable](a []T, o T) bool {
	for _, t := range a {
		if o == t {
			return true
		}
	}
	return false
}

package slicesx

func ContainsAny[T comparable](haystack []T, needle []T) bool {
	for _, n := range needle {
		for _, h := range haystack {
			if h == n {
				return true
			}
		}
	}
	return false
}

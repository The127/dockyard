package pointer

func Equal[T comparable](a, b *T) bool {
	if a == b {
		return true
	}
	if a != nil && b != nil {
		return *a == *b
	}
	return false
}

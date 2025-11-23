package repositories

// Changes is a generic type used to track changes in a model.
// T must conform to the comparable constraint to ensure consistent behavior with map keys.
type Changes[T comparable] struct {
	changes map[T]struct{}
}

// GetChanges returns a slice of all tracked changes in the Changes collection.
func (b *Changes[T]) GetChanges() []T {
	changes := make([]T, 0, len(b.changes))
	for change := range b.changes {
		changes = append(changes, change)
	}

	return changes
}

// ClearChanges removes all tracked changes by resetting the internal map to an empty state.
// This method is only supposed to be called by the repository implementations.
func (b *Changes[T]) ClearChanges() {
	b.changes = make(map[T]struct{})
}

// trackChange adds the given key to the changes map to indicate a modification of the corresponding property.
func (b *Changes[T]) trackChange(key T) {
	b.changes[key] = struct{}{}
}

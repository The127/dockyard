package queries

type PagedResponse[T any] struct {
	Items []T
}

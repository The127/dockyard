package handlers

type PagedResponse[T any] struct {
	Items []T `json:"items"`
}

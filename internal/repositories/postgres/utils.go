package postgres

type RowScanner interface {
	Scan(...interface{}) error
}

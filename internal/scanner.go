package internal

type Scanner interface {
	Next() (key []byte, row Row, ok bool, err error)
	Columns() []string
}

package internal

import "fmt"

type Pager interface {
}

type MemoryPager struct {
}

func NewPager(dsn string) (Pager, error) {
	if dsn == ":memory:" {
		return NewMemoryPager(), nil
	}

	return nil, fmt.Errorf("WIP")
}

func NewMemoryPager() *MemoryPager {
	return &MemoryPager{}
}

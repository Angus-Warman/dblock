package internal

import "fmt"

type Pager interface {
	Commit() error
	Rollback() error
}

type MemoryPager struct {
}

// Commit implements [Pager].
func (m *MemoryPager) Commit() error {
	return nil
}

// Rollback implements [Pager].
func (m *MemoryPager) Rollback() error {
	return nil
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

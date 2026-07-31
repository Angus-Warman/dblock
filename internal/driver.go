package internal

import (
	"context"
	"database/sql/driver"
	"fmt"
)

type Driver struct {
}

func NewDriver() driver.Driver {
	return &Driver{}
}

// Open implements [driver.Open].
func (d *Driver) Open(name string) (driver.Conn, error) {
	return NewConn(name)
}

type Conn struct {
	dsn string
}

func NewConn(dsn string) (*Conn, error) {
	return &Conn{
		dsn: dsn,
	}, nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	return NewTx(c.dsn)
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return NewStmt(query)
}

func (c *Conn) Close() error {
	return nil
}

func (c *Conn) Ping(ctx context.Context) error {
	return nil
}

type Tx struct {
	pager Pager
}

func NewTx(dsn string) (*Tx, error) {
	pager, err := NewPager(dsn)

	if err != nil {
		return nil, fmt.Errorf("NewTx: %w", err)
	}

	return &Tx{
		pager: pager,
	}, nil
}

func (t *Tx) Commit() error {
	return nil
}

func (t *Tx) Rollback() error {
	return nil
}

type Stmt struct {
}

func NewStmt(query string) (*Stmt, error) {
	return &Stmt{}, nil
}

func (s *Stmt) NumInput() int {
	return -1
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return &Result{}, nil
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return &Rows{}, nil
}

func (s *Stmt) Close() error {
	return nil
}

type Result struct {
}

// LastInsertId implements [driver.Result].
func (r *Result) LastInsertId() (int64, error) {
	return -1, nil
}

// RowsAffected implements [driver.Result].
func (r *Result) RowsAffected() (int64, error) {
	return -1, nil
}

type Rows struct {
}

// Close implements [driver.Rows].
func (r *Rows) Close() error {
	return nil
}

// Columns implements [driver.Rows].
func (r *Rows) Columns() []string {
	return []string{}
}

// Next implements [driver.Rows].
func (r *Rows) Next(dest []driver.Value) error {
	return nil
}

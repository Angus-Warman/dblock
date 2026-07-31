package internal

import (
	"context"
	"database/sql/driver"
	"dblock2/internal/parser"
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
	pager Pager
}

func NewConn(dsn string) (*Conn, error) {
	pager, err := NewPager(dsn)

	if err != nil {
		return nil, fmt.Errorf("NewTx: %w", err)
	}

	return &Conn{
		pager: pager,
	}, nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	return c.NewTx()
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
	conn *Conn
}

func (c *Conn) NewTx() (*Tx, error) {
	return &Tx{
		conn: c,
	}, nil
}

func (t *Tx) Commit() error {
	return t.conn.pager.Commit()
}

func (t *Tx) Rollback() error {
	return t.conn.pager.Rollback()
}

type Stmt struct {
}

func NewStmt(query string) (*Stmt, error) {
	p := parser.New()
	_ = p

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

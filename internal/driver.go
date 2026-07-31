package internal

import (
	"context"
	"database/sql/driver"
	"fmt"
)

// Driver implements [driver.Driver].
type Driver struct {
}

func NewDriver() driver.Driver {
	return &Driver{}
}

// Open implements [driver.Open].
func (d *Driver) Open(name string) (driver.Conn, error) {
	return NewConn(name)
}

func (c *Conn) Ping(ctx context.Context) error {
	return nil
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
	return &Tx{}, nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return NewStmt(query)
}

func (c *Conn) Close() error {
	return nil
}

type Tx struct {
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

func (c *Stmt) NumInput() int {
	return -1
}

func (c *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("WIP")
}

func (c *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, fmt.Errorf("wip")
}

func (c *Stmt) Close() error {
	return nil
}

package internal

import (
	"context"
	"database/sql/driver"
	"dblock2/internal/parser"
	"fmt"
	"io"
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
	pager    Pager // Shared pager
	activeTx *Tx
}

func NewConn(dsn string) (*Conn, error) {
	pager, err := NewPager(dsn)

	if err != nil {
		return nil, fmt.Errorf("NewConn: %w", err)
	}

	return &Conn{
		pager: pager,
	}, nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	if c.activeTx != nil {
		return nil, fmt.Errorf("a transaction already exists")
	}

	tx, err := c.NewTx()

	if err != nil {
		return nil, fmt.Errorf("Begin: %w", err)
	}

	c.activeTx = tx

	return tx, nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.NewStmt(query)
}

func (c *Conn) Close() error {
	return c.pager.Close()
}

func (c *Conn) Ping(ctx context.Context) error {
	return nil
}

type Tx struct {
	conn    *Conn
	txPager *TxPager
}

func (c *Conn) NewTx() (*Tx, error) {
	return &Tx{
		conn:    c,
		txPager: NewTxPager(c.pager),
	}, nil
}

func (t *Tx) Commit() error {
	err := t.txPager.Commit()

	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	t.conn.activeTx = nil
	return nil
}

func (t *Tx) Rollback() error {
	err := t.txPager.Rollback()

	if err != nil {
		return fmt.Errorf("rollback: %w", err)
	}

	t.conn.activeTx = nil
	return nil
}

type Stmt struct {
	conn      *Conn
	execStmt  *ExecStmt
	queryStmt *QueryStmt
}

func (c *Conn) NewStmt(query string) (*Stmt, error) {
	parsed, err := parser.Parse(query)

	if err != nil {
		return nil, err
	}

	stmt := &Stmt{
		conn: c,
	}

	if parsed.Create != nil {
		exec, err := resolveCreate(parsed.Create)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = exec
	}

	if parsed.Insert != nil {
		exec, err := resolveInsert(parsed.Insert)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = exec
	}

	if parsed.Select != nil {
		stmt.queryStmt = &QueryStmt{
			tableName: parsed.Select.TableName,
		}
	}

	return stmt, nil
}

func (s *Stmt) NumInput() int {
	return -1
}

func getEngineArgs(args []driver.Value) []any {
	engineArgs := []any{}

	for _, arg := range args {
		engineArgs = append(engineArgs, arg)
	}

	return engineArgs
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	implicitTx := s.conn.activeTx == nil
	if implicitTx {
		_, err := s.conn.Begin()

		if err != nil {
			return nil, fmt.Errorf("Exec: %w", err)
		}
	}

	e, err := NewEngine(s.conn.activeTx.txPager)

	if err != nil {
		return nil, err
	}

	engineArgs := getEngineArgs(args)

	lastInsertID, rowsAffected, err := e.Exec(s.execStmt, engineArgs)

	if err != nil {
		return nil, fmt.Errorf("Exec: %w", err)
	}

	if implicitTx {
		err = s.conn.activeTx.Commit()

		if err != nil {
			return nil, fmt.Errorf("Exec: %w", err)
		}
	}

	return &Result{
		lastInsertID: lastInsertID,
		rowsAffected: rowsAffected,
	}, nil
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	implicitTx := s.conn.activeTx == nil
	if implicitTx {
		_, err := s.conn.Begin()

		if err != nil {
			return nil, fmt.Errorf("Exec: %w", err)
		}
	}

	e, err := NewEngine(s.conn.activeTx.txPager)

	if err != nil {
		return nil, err
	}

	engineArgs := getEngineArgs(args)

	scanner, err := e.Query(s.queryStmt, engineArgs)

	if err != nil {
		return nil, err
	}

	if implicitTx {
		err = s.conn.activeTx.Commit()

		if err != nil {
			return nil, fmt.Errorf("Exec: %w", err)
		}
	}

	return &Rows{
		scanner: scanner,
	}, nil
}

func (s *Stmt) Close() error {
	return nil
}

type Result struct {
	lastInsertID int64
	rowsAffected int64
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
	scanner Scanner
}

// Close implements [driver.Rows].
func (r *Rows) Close() error {
	return nil
}

// Columns implements [driver.Rows].
func (r *Rows) Columns() []string {
	return r.scanner.Columns()
}

// Next implements [driver.Rows].
func (r *Rows) Next(dest []driver.Value) error {
	_, row, ok, err := r.scanner.Next()
	if err != nil {
		return err
	}
	if !ok {
		return io.EOF
	}

	for i, v := range row.Values {
		if i < len(dest) {
			dest[i] = v
		}
	}
	return nil
}

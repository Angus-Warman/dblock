package internal

import (
	"context"
	"database/sql/driver"
	"dblock2/internal/parser"
	"fmt"
	"io"

	"github.com/google/uuid"
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

func (d *Driver) OpenConnector(dsn string) (driver.Connector, error) {
	source, err := NewPagerSource(dsn)

	if err != nil {
		return nil, err
	}

	return &Connector{
		dsn:    dsn,
		source: source,
	}, nil
}

type Connector struct {
	dsn    string
	source *PagerSource
}

func (c *Connector) Connect(_ context.Context) (driver.Conn, error) {
	return &Conn{
		source: c.source,
	}, nil
}

func (c *Connector) Driver() driver.Driver {
	return &Driver{}
}

type Conn struct {
	source   *PagerSource // Shared between all tx
	activeTx *Tx
}

func NewConn(dsn string) (*Conn, error) {
	source, err := NewPagerSource(dsn)

	if err != nil {
		return nil, fmt.Errorf("NewConn: %w", err)
	}

	return &Conn{
		source: source,
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

func (c *Conn) CheckNamedValue(nv *driver.NamedValue) error {
	switch nv.Value.(type) {
	case uuid.UUID:
		return nil // Leave intact
	default:
		return driver.ErrSkip
	}
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.NewStmt(query)
}

func (c *Conn) Close() error {
	return c.source.Close()
}

func (c *Conn) Ping(ctx context.Context) error {
	return nil
}

type Tx struct {
	conn  *Conn
	pager *StoragePager
}

func (c *Conn) NewTx() (*Tx, error) {
	return &Tx{
		conn:  c,
		pager: c.source.Begin(),
	}, nil
}

func (t *Tx) Commit() error {
	err := t.pager.Commit()

	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	t.conn.activeTx = nil
	return nil
}

func (t *Tx) Rollback() error {
	err := t.pager.Rollback()

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
		create, err := resolveCreate(parsed.Create)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = create
	}

	if parsed.Insert != nil {
		insert, err := resolveInsert(parsed.Insert)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = insert
	}

	if parsed.Drop != nil {
		drop, err := resolveDrop(parsed.Drop)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = drop
	}

	if parsed.Select != nil {
		sel, err := resolveSelect(parsed.Select)

		if err != nil {
			return nil, err
		}

		stmt.queryStmt = sel
	}

	if parsed.Update != nil {
		update, err := resolveUpdate(parsed.Update)

		if err != nil {
			return nil, err
		}

		stmt.execStmt = update
	}

	if parsed.Pragma != nil {
		p, err := resolvePragma(parsed.Pragma)

		if err != nil {
			return nil, err
		}

		if p.value != "" {
			stmt.execStmt = &ExecStmt{pragmaStmt: p}
		} else {
			stmt.queryStmt = &QueryStmt{pragmaStmt: p}
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

	e, err := NewEngine(s.conn.activeTx.pager)

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

	e, err := NewEngine(s.conn.activeTx.pager)

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

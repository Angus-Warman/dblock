package parser

import "fmt"

type Parser struct {
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(query string) (*ExecStmt, *QueryStmt, error) {
	return nil, nil, fmt.Errorf("WIP")
}

type ExecStmt struct {
}

type QueryStmt struct {
}

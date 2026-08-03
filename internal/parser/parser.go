package parser

import "fmt"

func Parse(query string) (*ParsedStmt, error) {
	s, err := sqlParser.ParseString("", query)

	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return s, nil
}

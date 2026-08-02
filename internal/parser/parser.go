package parser

func Parse(query string) (*ParsedStmt, error) {
	return sqlParser.ParseString("", query)
}

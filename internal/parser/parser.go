package parser

func Parse(query string) (*ParsedStmt, error) {
	return sqlParser.ParseString("", query)

	// if err != nil {
	// 	return nil, nil, err
	// }

	// if parsed.Create != nil {
	// 	return &internal.ExecStmt{
	// 		tableName: parsed.Create.Name,
	// 	}, nil, nil
	// }

	// return nil, nil, fmt.Errorf("WIP")
}

package internal

import "fmt"

func (e *Engine) querySelect(stmt *SelectStmt, args []any) (Scanner, error) {
	scanner, err := e.createBaseScanner(stmt.tableName)

	if err != nil {
		return nil, err
	}

	for _, join := range stmt.joins {
		right, err := e.CreateFullScanner(join.tableName)

		if err != nil {
			return nil, err
		}

		scanner, err = NewJoinScanner(scanner, right, stmt, &join, args)

		if err != nil {
			return nil, err
		}
	}

	if stmt.where != nil {
		scanner, err = NewFilterScanner(scanner, stmt.where, args)

		if err != nil {
			return nil, err
		}
	}

	if hasAggregates(stmt) {
		scanner, err = NewAggregateScanner(scanner, stmt, args)

		if err != nil {
			return nil, err
		}
	}

	if stmt.orders != nil {
		scanner, err = NewOrderScanner(scanner, stmt, args)

		if err != nil {
			return nil, err
		}
	}

	if hasAggregates(stmt) {
		return scanner, nil
	}

	scanner, err = NewProjectorScanner(scanner, stmt, args)

	if err != nil {
		return nil, err
	}

	return scanner, nil
}

func (e *Engine) createBaseScanner(tableName string) (Scanner, error) {
	// Rudimentary, if an index exists, use it
	indexes, err := e.findIndexes(tableName)

	if err != nil {
		return nil, err
	}

	if len(indexes) > 0 {
		index := indexes[0]
		return e.CreateIndexScanner(tableName, index)
	}

	return e.CreateFullScanner(tableName)
}

func (e *Engine) CreateIndexScanner(tableName string, index *Index) (Scanner, error) {
	info, err := e.lookupTable(tableName)

	if err != nil {
		return nil, fmt.Errorf("create full scanner: %w", err)
	}

	tree := NewBtree(e.pager, info.rootPage)

	return NewIndexScanner(index, tree, info.table)
}

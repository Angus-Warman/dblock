package dblock2

import (
	"database/sql"
	"dblock2/internal"
)

func init() {
	sql.Register("dblock", internal.NewDriver())
}

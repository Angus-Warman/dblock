package dblock

import (
	"database/sql"

	"github.com/Angus-Warman/dblock/internal"
)

func init() {
	sql.Register("dblock", internal.NewDriver())
}

package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"

	_ "dblock2"
)

func main() {
	fmt.Println("Starting benchmark...")

	db, err := sql.Open("dblock", "bench.db")
	check(err)
	defer db.Close()

	db.SetMaxOpenConns(1)

	sizes := []int{10, 100, 1000, 10000}

	for _, n := range sizes {
		runBench(db, n)
	}
}

func runBench(db *sql.DB, n int) {
	_, err := db.Exec("DROP TABLE IF EXISTS foo")
	check(err)
	_, err = db.Exec("CREATE TABLE foo (id INTEGER, val TEXT)")
	check(err)

	tx, err := db.Begin()
	check(err)
	stmt, err := tx.Prepare("INSERT INTO foo (id, val) VALUES (?, ?)")
	check(err)
	for i := 1; i <= n; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("val-%d", i))
		check(err)
	}
	check(stmt.Close())
	check(tx.Commit())

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	lats := make([]time.Duration, 0, n)

	iters := min(n, 1000)

	for range iters {
		id := rng.Intn(n) + 1
		var gotID int
		var val string

		start := time.Now()
		err := db.QueryRow("SELECT * FROM foo WHERE id = ?", id).Scan(&gotID, &val)
		elapsed := time.Since(start)
		check(err)

		lats = append(lats, elapsed)
	}

	printLatencyStats(n, lats)
}

func printLatencyStats(n int, lats []time.Duration) {
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	p := func(pct float64) time.Duration {
		idx := int(pct * float64(len(lats)-1))
		return lats[idx]
	}
	fmt.Printf("N=%-6d p50=%v p95=%v p99=%v max=%v\n",
		n, p(0.50), p(0.95), p(0.99), lats[len(lats)-1])
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

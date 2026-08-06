package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	_ "dblock2"
)

func main() {
	fmt.Println("Starting...")

	db, err := sql.Open("dblock", "data.db")
	check(err)
	defer db.Close()

	db.SetMaxOpenConns(1)

	_, err = db.Exec("DROP TABLE IF EXISTS accounts")
	check(err)
	_, err = db.Exec("DROP TABLE IF EXISTS transactions")
	check(err)

	_, err = db.Exec("CREATE TABLE accounts (owner TEXT, balance INTEGER)")
	check(err)
	_, err = db.Exec(`CREATE TABLE transactions (from_account TEXT, to_account TEXT, amount INTEGER)`)
	check(err)

	const (
		numAccounts           = 100
		numWorkers            = 10
		numReaders            = 4
		startingBalance       = 1000
		transfersPerGoroutine = 50
		testDuration          = 30 * time.Second
	)
	accountSum := numAccounts * startingBalance

	// --- seed accounts ---
	seedTx, err := db.Begin()
	check(err)
	stmt, err := seedTx.Prepare("INSERT INTO accounts (owner, balance) VALUES (?, ?)")
	check(err)
	for i := range numAccounts {
		_, err = stmt.Exec(fmt.Sprintf("acct-%d", i), startingBalance)
		check(err)
	}
	check(stmt.Close())
	check(seedTx.Commit())

	var (
		writerWG      sync.WaitGroup
		readerWG      sync.WaitGroup
		committed     int64
		aborted       int64
		deadlocked    int64
		readViolation int64
		stopReaders   = make(chan struct{})
	)

	rngPool := sync.Pool{New: func() any { return rand.New(rand.NewSource(time.Now().UnixNano())) }}

	// --- writer goroutines: random transfers, some intentionally rolled back ---
	for id := range numWorkers {
		writerWG.Go(func() {
			rng := rngPool.Get().(*rand.Rand)
			defer rngPool.Put(rng)

			for j := range transfersPerGoroutine {
				from := fmt.Sprintf("acct-%d", rng.Intn(numAccounts))
				to := fmt.Sprintf("acct-%d", rng.Intn(numAccounts))
				if from == to {
					continue
				}
				amount := rng.Intn(50) + 1
				fmt.Printf("%v [%v/%v] transfer: %v from %v to %v\n", id, j+1, transfersPerGoroutine, amount, from, to)
				forceRollback := rng.Intn(10) == 0 // 10% intentional rollback

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := doTransfer(ctx, db, from, to, amount, forceRollback)
				cancel()

				switch {
				case err == nil:
					atomic.AddInt64(&committed, 1)
				case ctx.Err() == context.DeadlineExceeded:
					// Treat a timeout as a probable deadlock/livelock rather
					// than a normal abort — fail loudly, don't just count it.
					atomic.AddInt64(&deadlocked, 1)
					fmt.Printf("worker %d: transfer timed out (possible deadlock): %v\n", id, err)
				default:
					fmt.Println(err)
					atomic.AddInt64(&aborted, 1)
				}
			}
		})
	}

	// --- reader goroutines: SUM(balance) must always equal the invariant ---
	for range numReaders {
		readerWG.Go(func() {
			for {
				select {
				case <-stopReaders:
					return
				default:
				}
				var sum int
				err := db.QueryRow("SELECT SUM(balance) FROM accounts").Scan(&sum)
				if err != nil {
					fmt.Println("reader error:", err)
					continue
				}
				if sum != accountSum {
					atomic.AddInt64(&readViolation, 1)
					fmt.Printf("INVARIANT VIOLATION: sum=%d want=%d\n", sum, accountSum)
				}
				time.Sleep(time.Millisecond)
			}
		})
	}

	// Safety net: don't let a real deadlock hang the test suite forever.
	done := make(chan struct{})
	go func() {
		writerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testDuration):
		fmt.Println("TEST TIMED OUT — likely deadlock or livelock in engine")
		os.Exit(1)
	}
	close(stopReaders)
	readerWG.Wait()

	// --- final invariant check ---
	var finalSum int
	check(db.QueryRow("SELECT SUM(balance) FROM accounts").Scan(&finalSum))

	fmt.Printf("committed=%d aborted=%d deadlocked=%d readViolations=%d\n",
		committed, aborted, deadlocked, readViolation)
	fmt.Printf("final sum=%d expected=%d\n", finalSum, accountSum)

	if finalSum != accountSum {
		fmt.Println("FAIL: final balance sum does not match expected total")
		os.Exit(1)
	}
	if readViolation > 0 {
		fmt.Println("FAIL: mid-flight SUM(balance) invariant was violated — isolation guarantee broken")
		os.Exit(1)
	}
	if deadlocked > 0 {
		fmt.Println("FAIL: one or more transfers timed out, indicating deadlock/livelock")
		os.Exit(1)
	}
	fmt.Println("PASS")
}

// doTransfer runs a single transfer as one transaction: read both balances,
// verify funds, write both, commit. If forceRollback is set, it rolls back
// after writing (to test that rollback actually undoes the write and that
// concurrent readers never observe the uncommitted intermediate state).
func doTransfer(ctx context.Context, db *sql.DB, from, to string, amount int, forceRollback bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op if already committed

	var fromBalance int
	if err := tx.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE owner = ?", from).Scan(&fromBalance); err != nil {
		return err
	}
	if fromBalance < amount {
		return nil // insufficient funds, not an error — just skip
	}

	if _, err := tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - ? WHERE owner = ?", amount, from); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + ? WHERE owner = ?", amount, to); err != nil {
		return err
	}

	if forceRollback {
		return tx.Rollback()
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO transactions (from_account, to_account, amount) VALUES (?, ?, ?)`, from, to, amount); err != nil {
		return err
	}

	return tx.Commit()
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

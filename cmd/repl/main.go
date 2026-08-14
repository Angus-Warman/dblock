package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	_ "github.com/Angus-Warman/dblock"
)

var quitTerms = []string{
	"exit", "quit", ".exit", ".quit",
}

func main() {
	var dsn string

	if len(os.Args) > 1 {
		dsn = os.Args[1]
	} else {
		fmt.Println("opening in-memory DB")
		dsn = ":memory:"
	}

	db, err := openDB(dsn)

	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening db: ", err)
		os.Exit(1)
	}

	defer db.Close()

	fmt.Println("connected, ctrl+c or 'exit' to exit")

	runLoopWithIntercept(db)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("dblock", dsn)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func runLoopWithIntercept(db *sql.DB) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		runLoop(db)
		close(done)
	}()

	select {
	case <-quit:
		fmt.Println("\ngoodbye")
	case <-done:
	}
}

func runLoop(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\ngoodbye")
			break // EOF / Ctrl-D
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if slices.Contains(quitTerms, line) {
			break
		}

		start := time.Now()

		res, err := eval(db, line)

		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else {
			fmt.Printf("%v\n", res)
		}

		fmt.Printf("(%s)\n", time.Since(start))
	}
}

func eval(db *sql.DB, line string) (string, error) {
	if isQuery(line) {
		return evalQuery(db, line)
	} else {
		return evalExec(db, line)
	}
}

func isQuery(stmt string) bool {
	if strings.HasPrefix(stmt, "SELECT") {
		return true
	}

	if strings.HasPrefix(stmt, "PRAGMA") {
		parts := strings.Split(stmt, " ")

		if len(parts) == 2 {
			return true
		}
	}

	return false
}

func evalQuery(db *sql.DB, stmt string) (string, error) {
	rows, err := db.Query(stmt)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	headerString := strings.Join(cols, " | ")
	sb.WriteString(headerString)
	sb.WriteString("\n")
	underline := strings.Repeat("-", len(headerString))
	sb.WriteString(underline)
	sb.WriteString("\n")

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	count := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		parts := make([]string, len(cols))
		for i, v := range vals {
			parts[i] = fmt.Sprintf("%v", v)
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
		count++
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	plural := pluralise(count)

	fmt.Fprintf(&sb, "%d row%v", count, plural)
	return sb.String(), nil
}

func pluralise(num int) string {
	if num == 1 {
		return ""
	}

	return "s"
}

func evalExec(db *sql.DB, stmt string) (string, error) {
	res, err := db.Exec(stmt)
	if err != nil {
		return "", err
	}
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()

	plural := pluralise(int(affected))

	return fmt.Sprintf("ok, %d row%v affected, last id: %d", affected, plural, lastID), nil
}

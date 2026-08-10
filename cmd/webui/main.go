package main

import (
	"database/sql"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/Angus-Warman/dblock"
)

var db *sql.DB

//go:embed index.tmpl
var indexTemplate string
var tmpl = template.Must(template.New("index").Parse(indexTemplate))

type PageData struct {
	Query    string
	Error    string
	Columns  []string
	Rows     [][]any
	Duration time.Duration
	Tables   []string
}

func main() {
	var err error

	dsn := ":memory:"

	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}

	db, err = sql.Open("dblock", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", handleIndex)

	log.Println("Listening on http://localhost:8585")
	log.Fatal(http.ListenAndServe(":8585", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := PageData{}

	isPOST := r.Method == http.MethodPost
	isHTMX := r.Header.Get("HX-Request") != ""

	if isPOST {
		data.Query = r.FormValue("query")

		start := time.Now()
		err := executeQuery(&data)

		if err != nil {
			data.Error = err.Error()
		} else {
			data.Error = ""
		}

		data.Duration = time.Since(start)
	}

	tables, err := tableNames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.Tables = tables

	if isPOST && isHTMX {
		if err := tmpl.ExecuteTemplate(w, "result", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "tables-oob", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	if err := tmpl.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func executeQuery(data *PageData) error {
	if !isQuery(data.Query) {
		if _, err := db.Exec(data.Query); err != nil {
			return err
		}
		return nil
	}

	rows, err := db.Query(data.Query)

	if err != nil {
		return err
	}

	defer rows.Close()

	data.Columns, err = rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		values := make([]any, len(data.Columns))
		dest := make([]any, len(data.Columns))

		for i := range values {
			dest[i] = &values[i]
		}

		if err := rows.Scan(dest...); err != nil {
			return err
		}

		// Convert []byte -> string for nicer display.
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		data.Rows = append(data.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func tableNames() ([]string, error) {
	rows, err := db.Query("SELECT name FROM dblock_schema WHERE object_type = 'table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

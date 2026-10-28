package main

import (
	"database/sql"
	"fmt"
	"runtime"

	_ "turso.tech/database/tursogo"
)

func main() {
	db, err := sql.Open("turso", ":memory:")
	check(err)
	defer db.Close()
	// Open alone is lazy. Force the native library to execute SQL, including
	// integer, text and floating-point argument/result marshalling.
	_, err = db.Exec("CREATE TABLE sample (id INTEGER PRIMARY KEY, text TEXT, value REAL)")
	check(err)
	_, err = db.Exec("INSERT INTO sample VALUES (?, ?, ?)", 7, "purego ABI", 1.5)
	check(err)
	runtime.GC()
	var text string
	var value float64
	check(db.QueryRow("SELECT text, value FROM sample WHERE id = ?", 7).Scan(&text, &value))
	if text != "purego ABI" || value != 1.5 {
		panic(fmt.Sprintf("query: %q, %g", text, value))
	}
	check(db.Close())
	fmt.Println("Turso native query and values verified")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

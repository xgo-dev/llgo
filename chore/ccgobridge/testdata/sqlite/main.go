package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"runtime"

	"modernc.org/sqlite"
)

func main() {
	// These closures share code but must keep distinct captured environments.
	registerAdder("add_seven", 7)
	registerAdder("add_nine", 9)
	runtime.GC()

	db, err := sql.Open("sqlite", ":memory:")
	check(err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE sample (id INTEGER PRIMARY KEY, text TEXT); INSERT INTO sample VALUES (7, 'asm ABI0')")
	check(err)
	var text string
	check(db.QueryRow("SELECT text FROM sample WHERE id = ?", 7).Scan(&text))
	if text != "asm ABI0" {
		panic(text)
	}

	tx, err := db.Begin()
	check(err)
	_, err = tx.Exec("INSERT INTO sample VALUES (8, 'rollback')")
	check(err)
	check(tx.Rollback())
	var count int
	check(db.QueryRow("SELECT count(*) FROM sample").Scan(&count))
	if count != 1 {
		panic(count)
	}

	var seven, nine int64
	check(db.QueryRow("SELECT add_seven(35), add_nine(33)").Scan(&seven, &nine))
	if seven != 42 || nine != 42 {
		panic(fmt.Sprintf("callback environments: %d, %d", seven, nine))
	}
	check(db.Close())
	runtime.GC()
	fmt.Println("sqlite query verified")
	fmt.Println("sqlite rollback and captured UDF callbacks verified")
}

func registerAdder(name string, delta int64) {
	check(sqlite.RegisterDeterministicScalarFunction(name, 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			runtime.GC()
			return args[0].(int64) + delta, nil
		}))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

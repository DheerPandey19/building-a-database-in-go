package main

import (
	"fmt"
	"syscall"
)

func main() {
	// ---------------------------------------------------------
	// FIRST OPEN
	// ---------------------------------------------------------

	db := KV{
		Path: "database.db",
	}

	err := db.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Database opened")
	fmt.Println("Initial root:", db.tree.root)
	fmt.Println("Initial flushed pages:", db.page.flushed)

	// ---------------------------------------------------------
	// Insert a key/value
	// ---------------------------------------------------------

	err = db.Set(
		[]byte("apple"),
		[]byte("red"),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("Inserted apple")
	fmt.Println("Root:", db.tree.root)
	fmt.Println("Flushed pages:", db.page.flushed)
	fmt.Println("Temporary pages:", len(db.page.temp))

	// ---------------------------------------------------------
	// Read while database is open
	// ---------------------------------------------------------

	val := db.Get([]byte("apple"))
	fmt.Println("Value before closing:", string(val))

	// ---------------------------------------------------------
	// Close first database
	// ---------------------------------------------------------

	syscall.Close(db.fd)

	fmt.Println("Database closed")

	// ---------------------------------------------------------
	// SECOND OPEN
	// ---------------------------------------------------------

	db2 := KV{
		Path: "database.db",
	}

	err = db2.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Database reopened")
	fmt.Println("Loaded root:", db2.tree.root)
	fmt.Println("Loaded flushed pages:", db2.page.flushed)

	// ---------------------------------------------------------
	// Read after reopening
	// ---------------------------------------------------------

	val = db2.Get([]byte("apple"))

	fmt.Println("Value after reopening:", string(val))

	// ---------------------------------------------------------
	// Close second database
	// ---------------------------------------------------------

	syscall.Close(db2.fd)

	fmt.Println("Database closed again")
}
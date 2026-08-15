package main

import (
	"fmt"
	"syscall"
)

func main() {
	// ---------------------------------------------------------
	// Open database
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
	// Read the value back
	// ---------------------------------------------------------

	val := db.Get([]byte("apple"))

	fmt.Println("Value for apple:", string(val))

	// ---------------------------------------------------------
	// Try a key that doesn't exist
	// ---------------------------------------------------------

	val = db.Get([]byte("banana"))

	if val == nil {
		fmt.Println("banana not found")
	} else {
		fmt.Println("Value for banana:", string(val))
	}

	// ---------------------------------------------------------
	// Close database
	// ---------------------------------------------------------

	syscall.Close(db.fd)

	fmt.Println("Database closed")
}
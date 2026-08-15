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
	// Create a fake B+Tree node
	// ---------------------------------------------------------

	node := make([]byte, BTREE_PAGE_SIZE)

	// Give the node some recognizable data.
	node[0] = 123

	// Put the node into the temporary page list.
	ptr := db.pageAppend(node)

	fmt.Println("New page pointer:", ptr)
	fmt.Println("Temporary pages:", len(db.page.temp))

	// Make this new page the B+Tree root.
	db.tree.root = ptr

	fmt.Println("New root:", db.tree.root)

	// ---------------------------------------------------------
	// Write everything to disk
	// ---------------------------------------------------------

	err = updateFile(&db)
	if err != nil {
		panic(err)
	}

	fmt.Println("Database update completed")
	fmt.Println("Flushed pages:", db.page.flushed)
	fmt.Println("Temporary pages:", len(db.page.temp))

	// ---------------------------------------------------------
	// Close database
	// ---------------------------------------------------------

	err = syscall.Close(db.fd)
	if err != nil {
		panic(err)
	}

	// ---------------------------------------------------------
	// Reopen database
	// ---------------------------------------------------------

	db2 := KV{
		Path: "database.db",
	}

	err = db2.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("\nDatabase reopened")
	fmt.Println("Loaded root:", db2.tree.root)
	fmt.Println("Loaded flushed pages:", db2.page.flushed)

	// ---------------------------------------------------------
	// Read the root page back
	// ---------------------------------------------------------

	rootPage := db2.pageRead(db2.tree.root)

	fmt.Println("Root page first byte:", rootPage[0])

	// ---------------------------------------------------------
	// Close
	// ---------------------------------------------------------

	syscall.Close(db2.fd)
}
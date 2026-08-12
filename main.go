package main

import "fmt"

func main() {
	// -------------------------------------------------------------------------
	// Open database
	// -------------------------------------------------------------------------

	db := KV{
		Path: "database.db",
	}

	err := db.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Database opened successfully")

	// -------------------------------------------------------------------------
	// Create mmap
	// -------------------------------------------------------------------------

	err = extendMmap(&db, 4096)
	if err != nil {
		panic(err)
	}

	fmt.Println("mmap created successfully")
	fmt.Println("Mapped bytes:", db.mmap.total)
	fmt.Println("Number of mappings:", len(db.mmap.chunks))

	// -------------------------------------------------------------------------
	// Test pageRead
	// -------------------------------------------------------------------------

	// Ask for page number 2.
	// Each page is 4096 bytes.
	page := db.pageRead(2)

	fmt.Println("Page size:", len(page))
}
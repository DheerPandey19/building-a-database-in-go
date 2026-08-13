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

	// -------------------------------------------------------------------------
	// Test pageAppend
	// -------------------------------------------------------------------------

	node1 := make([]byte, BTREE_PAGE_SIZE)

	ptr1 := db.pageAppend(node1)

	fmt.Println("First page:", ptr1)
	fmt.Println("Temporary pages:", len(db.page.temp))

	node2 := make([]byte, BTREE_PAGE_SIZE)

	ptr2 := db.pageAppend(node2)

	fmt.Println("Second page:", ptr2)
	fmt.Println("Temporary pages:", len(db.page.temp))
}
package main

import "fmt"

func main() {
	db := KV{
		Path: "database.db",
	}
	
	err := db.Open()
	if err != nil {
		panic(err)
	}
	
	fmt.Println("Database opened successfully")
	
	err = extendMmap(&db, 4096)
	if err != nil {
		panic(err)
	}
	
	fmt.Println("mmap created successfully")
	fmt.Println("Mapped bytes:", db.mmap.total)
	fmt.Println("Number of mappings:", len(db.mmap.chunks))
}

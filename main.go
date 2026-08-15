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
    fmt.Println("Root:", db.tree.root)
    fmt.Println("Flushed pages:", db.page.flushed)
}
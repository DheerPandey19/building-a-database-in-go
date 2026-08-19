package main

import (
    "bytes"
    "fmt"
    "os"
    "syscall"
)

func main() {
	dbPath := "test.db"

	// Start from a clean database for this test.
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	// First session: create and write.
	db := &KV{Path: dbPath}
	if err := db.Open(); err != nil {
		panic(err)
	}

	if err := db.Set([]byte("name"), []byte("Dheer")); err != nil {
		panic(err)
	}
	if err := db.Set([]byte("language"), []byte("Go")); err != nil {
		panic(err)
	}
	if err := db.Set([]byte("project"), []byte("build-your-own-db")); err != nil {
		panic(err)
	}

	fmt.Println("Written values:")
	fmt.Println("name =", string(db.Get([]byte("name"))))
	fmt.Println("language =", string(db.Get([]byte("language"))))

	// Set() already fsyncs; close the descriptor before reopening.
	if err := syscall.Close(db.fd); err != nil {
		panic(err)
	}

	// Second session: prove values survived reopening.
	reopened := &KV{Path: dbPath}
	if err := reopened.Open(); err != nil {
		panic(err)
	}

	check := func(key, want string) {
		got := reopened.Get([]byte(key))
		if !bytes.Equal(got, []byte(want)) {
			panic(fmt.Sprintf(
				"persistence test failed: key=%q got=%q want=%q",
				key, got, want,
			))
		}
		fmt.Printf("OK: %s = %s\n", key, got)
	}

	fmt.Println("\nAfter reopening:")
	check("name", "Dheer")
	check("language", "Go")
	check("project", "build-your-own-db")

	// Update an existing key.
	if err := reopened.Set([]byte("language"), []byte("Golang")); err != nil {
		panic(err)
	}

	if err := syscall.Close(reopened.fd); err != nil {
		panic(err)
	}

	// Third session: prove the update also survived reopening.
	updated := &KV{Path: dbPath}
	if err := updated.Open(); err != nil {
		panic(err)
	}
	defer syscall.Close(updated.fd)

	got := updated.Get([]byte("language"))
	if !bytes.Equal(got, []byte("Golang")) {
		panic(fmt.Sprintf("update test failed: got=%q", got))
	}

	if updated.Get([]byte("missing-key")) != nil {
		panic("missing-key should not exist")
	}

	fmt.Println("\nAll persistence tests passed.")
}
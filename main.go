package main

import "fmt"

func main() {
	// -------------------------------------------------------------------------
	// Simple in-memory page manager
	// -------------------------------------------------------------------------

	pages := make(map[uint64][]byte)
	var nextPage uint64 = 1

	tree := BTree{
		get: func(ptr uint64) []byte {
			return pages[ptr]
		},

		new: func(node []byte) uint64 {
			ptr := nextPage
			nextPage++

			pages[ptr] = node

			return ptr
		},

		del: func(ptr uint64) {
			delete(pages, ptr)
		},
	}

	// -------------------------------------------------------------------------
	// Basic inserts
	// -------------------------------------------------------------------------

	fmt.Println("Inserting keys...")

	tree.Insert([]byte("apple"), []byte("red"))
	tree.Insert([]byte("banana"), []byte("yellow"))
	tree.Insert([]byte("cat"), []byte("animal"))

	// -------------------------------------------------------------------------
	// Basic gets
	// -------------------------------------------------------------------------

	fmt.Println("\nGetting values...")

	value := treeGet(&tree, []byte("apple"))
	fmt.Println("apple =", string(value))

	value = treeGet(&tree, []byte("banana"))
	fmt.Println("banana =", string(value))

	value = treeGet(&tree, []byte("cat"))
	fmt.Println("cat =", string(value))

	// -------------------------------------------------------------------------
	// Missing key
	// -------------------------------------------------------------------------

	value = treeGet(&tree, []byte("dog"))

	if value == nil {
		fmt.Println("dog = <not found>")
	} else {
		fmt.Println("dog =", string(value))
	}

	// -------------------------------------------------------------------------
	// Update an existing key
	// -------------------------------------------------------------------------

	fmt.Println("\nUpdating apple...")

	tree.Insert([]byte("apple"), []byte("green"))

	value = treeGet(&tree, []byte("apple"))
	fmt.Println("apple =", string(value))
}
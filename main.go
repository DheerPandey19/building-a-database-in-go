package main

import "fmt"

func main() {

	// Create one empty 4KB page.
	page := make(BNode, BTREE_PAGE_SIZE)

	// Initialize it.
	page.setHeader(BNODE_LEAF, 3)

	fmt.Println("========== HEADER ==========")
	fmt.Println("Type :", page.btype())
	fmt.Println("Keys :", page.nkeys())

	// Store some fake pointers.
	page.setPointer(0, 100)
	page.setPointer(1, 250)
	page.setPointer(2, 999)

	fmt.Println()

	fmt.Println("========== POINTERS ==========")

	for i := uint16(0); i < page.nkeys(); i++ {
		fmt.Printf("Pointer[%d] = %d\n", i, page.getPointer(i))
	}
}
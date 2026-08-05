package main

import "encoding/binary"

// -----------------------------------------------------------------------------
// Page Layout
// -----------------------------------------------------------------------------
//
// +---------------------------------------------------------------+
// | Header | Pointer Array | Offset Array | Key-Value Data | Free |
// +---------------------------------------------------------------+
//
// Header:
//
// +----------------------+
// | type (2B) | nkeys(2B)|
// +----------------------+
//
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

const (
	BNODE_NODE = 1 // Internal node
	BNODE_LEAF = 2 // Leaf node
)

const (
	BTREE_PAGE_SIZE = 4096
)

// -----------------------------------------------------------------------------
// BNode
// -----------------------------------------------------------------------------

// A B+Tree node stored directly as bytes.
// The entire slice represents one 4KB page.
type BNode []byte

// -----------------------------------------------------------------------------
// Header
// -----------------------------------------------------------------------------

// Returns the node type.
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

// Returns the number of keys.
func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

// Writes the header.
func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

// -----------------------------------------------------------------------------
// Pointer Array
// -----------------------------------------------------------------------------

// Returns the page number stored at index idx.
func (node BNode) getPointer(idx uint16) uint64 {

	if idx >= node.nkeys() {
		panic("pointer index out of bounds")
	}

	pos := 4 + idx*8

	return binary.LittleEndian.Uint64(node[pos:])
}

// Stores a page number at index idx.
func (node BNode) setPointer(idx uint16, value uint64) {

	if idx >= node.nkeys() {
		panic("pointer index out of bounds")
	}

	pos := 4 + idx*8

	binary.LittleEndian.PutUint64(node[pos:], value)
}
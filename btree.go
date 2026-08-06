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

// -----------------------------------------------------------------------------
// Offset Array
// -----------------------------------------------------------------------------

// Returns the relative offset (within the KV data section) where the idx-th
// KV pair begins. idx may range from 0 to nkeys (inclusive). idx == nkeys
// returns the offset just past the last KV pair.
func (node BNode) getOffset(idx uint16) uint16 {

	if idx > node.nkeys() {
		panic("offset index out of bounds")
	}

	if idx == 0 {
		return 0
	}

	pos := 4 + node.nkeys()*8 + 2*(idx-1)

	return binary.LittleEndian.Uint16(node[pos:])
}

// Stores the relative offset of the idx-th KV pair.
// Offset for idx == 0 is always 0 and is not stored.
func (node BNode) setOffset(idx uint16, offset uint16) {

	if idx > node.nkeys() {
		panic("offset index out of bounds")
	}

	if idx == 0 {
		return
	}

	pos := 4 + node.nkeys()*8 + 2*(idx-1)

	binary.LittleEndian.PutUint16(node[pos:], offset)
}

// -----------------------------------------------------------------------------
// Key-Value Access
// -----------------------------------------------------------------------------

// Returns the absolute byte position (within the page) where the idx-th
// KV pair begins.
func (node BNode) kvPos(idx uint16) uint16 {

	if idx > node.nkeys() {
		panic("KV index out of bounds")
	}

	// Layout:
	// Header | Pointer Array | Offset Array | KV Data
	//
	// kvPos = start of KV Data + relative offset of the idx-th KV.
	return 4 +
		8*node.nkeys() +
		2*node.nkeys() +
		node.getOffset(idx)
}

// Returns the key stored at index idx.
func (node BNode) getKey(idx uint16) []byte {

	if idx >= node.nkeys() {
		panic("key index out of bounds")
	}

	// Beginning of this KV pair.
	pos := node.kvPos(idx)

	// Read key length.
	klen := binary.LittleEndian.Uint16(node[pos:])

	// KV layout:
	// +----------+----------+------+------+
	// | key_size | val_size | key  | val  |
	// +----------+----------+------+------+
	//
	// Skip the 4-byte size header and return only the key bytes.
	return node[pos+4 : pos+4+klen]
}

// Returns the value stored at index idx.
func (node BNode) getVal(idx uint16) []byte {

	if idx >= node.nkeys() {
		panic("value index out of bounds")
	}

	// Beginning of this KV pair.
	pos := node.kvPos(idx)

	// Read the key and value lengths.
	klen := binary.LittleEndian.Uint16(node[pos:])
	vlen := binary.LittleEndian.Uint16(node[pos+2:])


	// Skip:
	//   4 bytes -> key_size + val_size
	//   klen    -> key bytes
	// to arrive at the beginning of the value.
	vStart := pos + 4 + klen

	return node[vStart : vStart+vlen]
}

// Returns the number of bytes currently occupied by this node.
func (node BNode) nbytes() uint16 {

	//last one's offset points to end

	// Page layout:
	//
	// +-----------------------------------------------------------+
	// | Header | Pointer Array | Offset Array | KV Data | Unused |
	// +-----------------------------------------------------------+
	//
	// Occupied bytes =
	//   Header
	// + Pointer Array
	// + Offset Array
	// + KV Data
	//
	// getOffset(nkeys()) returns the offset immediately after the
	// last KV pair, i.e. the total size of the KV data section.
	return 4 + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(node.nkeys())
}
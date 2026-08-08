package main

import ("bytes",
         "encoding/binary")

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

	
	return 4 + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(node.nkeys())
}



// -----------------------------------------------------------------------------
// Node Construction
// ----------------------------------------------------------------------------

func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {

	// -------------------------------------------------------------------------
	// 1. Store the pointer
	// -------------------------------------------------------------------------

	new.setPointer(idx, ptr)

	// -------------------------------------------------------------------------
	// 2. Find where this KV should start
	// -------------------------------------------------------------------------

	pos := new.kvPos(idx)

	// -------------------------------------------------------------------------
	// 3. Store key and value lengths
	// -------------------------------------------------------------------------

	binary.LittleEndian.PutUint16(new[pos:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+2:], uint16(len(val)))

	// -------------------------------------------------------------------------
	// 4. Store the key
	// -------------------------------------------------------------------------

	copy(new[pos+4:], key)

	// -------------------------------------------------------------------------
	// 5. Store the value
	// -------------------------------------------------------------------------

	copy(new[pos+4+uint16(len(key)):], val)

	// -------------------------------------------------------------------------
	// 6. Store the offset for the NEXT KV
	// -------------------------------------------------------------------------

	new.setOffset(
		idx+1,
		new.getOffset(idx)+4+uint16(len(key))+uint16(len(val)),
	)
}

// -----------------------------------------------------------------------------
// Copy a range of KV pairs from an old node into a new node
// -----------------------------------------------------------------------------
//
// new    -> destination node
// old    -> source node
// dstNew -> index in new where copying starts
// srcOld -> index in old where copying starts
// n      -> number of KV pairs to copy
//
// Example:
//
// old:  [k1] [k3] [k7]
//             ↑
//          srcOld = 1
//
// new:  [   ] [   ]
//        ↑
//     dstNew = 0
//
// nodeAppendRange(new, old, 0, 1, 2)
//
// copies:
// old[1] -> new[0]
// old[2] -> new[1]
//
func nodeAppendRange(
	new BNode,
	old BNode,
	dstNew uint16,
	srcOld uint16,
	n uint16,
) {
	for i := uint16(0); i < n; i++ {

		dst := dstNew + i
		src := srcOld + i

		// Copy the pointer, key, and value
		// using the existing nodeAppendKV helper.
		nodeAppendKV(
			new,
			dst,
			old.getPointer(src),
			old.getKey(src),
			old.getVal(src),
		)
	}
}

// -----------------------------------------------------------------------------
// Insert a new KV pair into a leaf node
// -----------------------------------------------------------------------------
//
// Creates a new leaf node from the old node with one additional KV pair.
//
// The insertion works in three steps:
//
// 1. Copy all KVs before idx
// 2. Insert the new KV at idx
// 3. Copy all remaining KVs after idx
//
// Example:
//
// old:  [k1] [k3] [k7]
//                 ^
//             insert k5 at idx=2
//
// new:  [k1] [k3] [k5] [k7]
//

func leafInsert(old BNode, new BNode, idx uint16, key []byte, value []byte) {
    new.setHeader(BNODE_LEAF, old.nkeys()+1)

    // Copy KVs before the insertion point.
    nodeAppendRange(new, old, 0, 0, idx)

    // Insert the new KV at idx.
    nodeAppendKV(new, idx, 0, key, value)

    // Copy the remaining KVs, shifted one position to the right.
    nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

func leafUpdate(old BNode, new BNode, idx uint16, key []byte, value []byte) {
    new.setHeader(BNODE_LEAF, old.nkeys())

    // Copy KVs before the update position.
    nodeAppendRange(new, old, 0, 0, idx)

    // Replace the existing KV at idx.
    nodeAppendKV(new, idx, 0, key, value)

    // Copy the remaining KVs without shifting them.
    nodeAppendRange(new, old, idx+1, idx+1, old.nkeys()-(idx+1))
}

// -----------------------------------------------------------------------------
// Find the last key <= the given key
// -----------------------------------------------------------------------------
//
// Returns the index of the largest key that is <= key.
//
// Example:
//
// node:   [k1] [k3] [k7]
//
// lookup k3 -> 1
// lookup k5 -> 1
// lookup k7 -> 2
// lookup k9 -> 2
//


func nodeLookupLE(node BNode,current[]byte) uint16{
	nkeys :=node.nkeys()

	var i uint16

	// 	bytes.Compare(a, b)
	// Result	Meaning
	// < 0	a < b
	// 0	a == b
	// > 0	a > b
	for i:=0;i<nkeys;i++{
		cmp := bytes.Compare(node.getKey(i),current)

		if cmp == 0{
			return i
		}
		if cmp > 0{
			return i-1
		}
		
	}
	return i-1
}
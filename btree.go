package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"os"
)
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
	DB_META_SIZE = BTREE_PAGE_SIZE // Size of the database metadata page.
)

// -----------------------------------------------------------------------------
// BNode
// -----------------------------------------------------------------------------

// A B+Tree node stored directly as bytes.
// The entire slice represents one 4KB page.
type BNode []byte

// -----------------------------------------------------------------------------
// BTree
// -----------------------------------------------------------------------------

// B+Tree structure.
// B+Tree does not directly know how pages are stored on disk.
// Instead, the B+Tree 3 functions (callbacks) that handle disk/page operations:
type BTree struct {
	// Root page number.
	root uint64

	// Read a node from a page number.
	get func(uint64) []byte

	// Allocate a new page containing the given node.
	new func([]byte) uint64

	// Delete/deallocate a page.
	del func(uint64)
}
// -----------------------------------------------------------------------------
// KV Store
// -----------------------------------------------------------------------------
// KV represents the persistent key-value database.
// It owns the database file and the B+Tree used to store the data.
type KV struct{

		// Path is the path to the database file on disk.
		// Database file path.
		Path string

		// Open file descriptor.
		// fd is the file descriptor for the opened database file.
		// It is used for reading, writing, and syncing the file.
		fd int

		// B+Tree stored in the database.
		tree BTree

		// mmap information
		mmap struct {
			total  int
			chunks [][]byte
		}

		// page allocation information
		page struct {
			flushed uint64
			temp    [][]byte
		}
}
func extendMmap(db *KV, size int) error {
	// The existing memory mappings are already large enough.
	if size <= db.mmap.total {
		return nil
	}

	// Allocate 64 MB for the first mapping.
	// After that, grow the mapping exponentially.
	alloc := db.mmap.total

	if alloc == 0 {
		alloc = 64 << 20 // 64 MB
	} else {
		alloc *= 2
	}

	// If the requested size is larger than the allocation,
	// keep doubling until the new mapping is large enough.
	for db.mmap.total+alloc < size {
		alloc *= 2
	}

	// Map this portion of the database file into memory.
	// The offset starts immediately after all existing mappings.
	chunk, err := syscall.Mmap(
		db.fd,
		int64(db.mmap.total),
		alloc,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}

	// Remember the new mapping.
	db.mmap.total += alloc
	db.mmap.chunks = append(db.mmap.chunks, chunk)

	return nil
}

// Open opens the database file, creating it if it does not exist.
// The file is opened for both reading and writing.
func (db *KV)Open() error{
	fd,err :=syscall.Open(db.Path,syscall.O_RDWR|os.O_CREATE,0644,)

	if err !=nil{
		return err
	}

	// Store the file descriptor so the database can use it
	// for subsequent page I/O operations.
	db.fd=fd
	return nil
}

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


func nodeLookupLE(node BNode, current []byte) uint16 {
	nkeys := node.nkeys()

	var i uint16

	for i = 0; i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), current)

		if cmp == 0 {
			return i
		}

		if cmp > 0 {
			return i - 1
		}
	}

	return i - 1
}

func leafInsertOrUpdate(new BNode,old BNode,key []byte,val []byte) {

	//finding the index of the last key <= the given key
    idx := nodeLookupLE(old, key)

    if bytes.Equal(key, old.getKey(idx)) {
        // Key already exists -> update its value.
        leafUpdate(old, new, idx, key, val)
    } else {
        // Key doesn't exist -> insert after the last key <= key.
        leafInsert(old, new, idx+1, key, val)
    }
}

// -----------------------------------------------------------------------------
// Split an oversized node into two nodes
// -----------------------------------------------------------------------------
//
// Splits old into left and right while making sure the right node fits
// within a single 4KB page.
//
// We start by splitting the keys roughly in half, then move the split
// position left or right depending on which side is too large.
//

func nodeSplit2(old BNode, left BNode, right BNode) {
    if old.nkeys() < 2 {
        panic("Cannot split a node with less than 2 keys")
    }

    // Start with the split in the middle.
    nleft := old.nkeys() / 2

    // Calculate the size of the left node.
    leftBytes := func() uint16 {
        return 4 + 8*nleft + 2*nleft + old.getOffset(nleft)
    }

    // If the left node is too large, move the split point left.
    for leftBytes() > BTREE_PAGE_SIZE {
        nleft--
    }

    if nleft < 1 {
        panic("left node cannot fit")
    }

    // Calculate the size of the right node.
    rightBytes := func() uint16 {
        return old.nbytes() - leftBytes() + 4
    }

    // If the right node is too large, move the split point right.
    for rightBytes() > BTREE_PAGE_SIZE {
        nleft++
    }

    if nleft >= old.nkeys() {
        panic("right node cannot fit")
    }

    // Number of keys in the right node.
    nright := old.nkeys() - nleft

    // Set headers.
    left.setHeader(old.btype(), nleft)
    right.setHeader(old.btype(), nright)

    // Copy left half.
    nodeAppendRange(left, old, 0, 0, nleft)

    // Copy right half.
    nodeAppendRange(right, old, 0, nleft, nright)
}

// -----------------------------------------------------------------------------
// Split an oversized node into 1, 2, or 3 nodes
// -----------------------------------------------------------------------------
//
// If the node already fits in one page, no split is needed.
//
// Otherwise:
//   1. Split old into left + right.
//   2. If left fits, return 2 nodes.
//   3. If left is still too large, split left again.
//
// This handles the case where a large KV near the middle of the node
// prevents the first split from producing two page-sized nodes.


func nodeSplit3(old BNode)(uint16,[3]BNode){

	// -------------------------------------------------------------------------
	// 1. Node already fits in one page
	// -------------------------------------------------------------------------

	if old.nbytes() <= BTREE_PAGE_SIZE { 
		// old may be backed by a 2-page (8192 byte) buffer because 
		// // treeInsert() creates temporary nodes with 2*BTREE_PAGE_SIZE. 
		// // 
		// // The node is logically one page, so return exactly one page. 
		return 1, [3]BNode{old[:BTREE_PAGE_SIZE]} }

		fmt.Println("SPLIT",old.nbytes(), "bytes")

	// -------------------------------------------------------------------------
	// 2. First split
	// -------------------------------------------------------------------------

	// Left may need to be split again, so give it 2 pages of space.
	left := BNode(make([]byte, 2*BTREE_PAGE_SIZE))

	// Right should fit into one page after nodeSplit2.
	right := BNode(make([]byte, BTREE_PAGE_SIZE))

	nodeSplit2(old,left,right)

	// -------------------------------------------------------------------------
	// 3. Check whether the left side fits
	// -------------------------------------------------------------------------

	if left.nbytes()<=BTREE_PAGE_SIZE{
		left = left[:BTREE_PAGE_SIZE]
		return 2,[3]BNode{left,right}
	}

	// -------------------------------------------------------------------------
	// 4. Left is still too large -> split it again
	// -------------------------------------------------------------------------

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))

	nodeSplit2(left,leftleft,middle)

	// leftleft must fit because nodeSplit2 guarantees the right side fits
	// and the remaining split is now small enough.
	if leftleft.nbytes() > BTREE_PAGE_SIZE {
		panic("left-left node is too large")
	}

	// -------------------------------------------------------------------------
	// 5. Return three nodes
	// -------------------------------------------------------------------------

	return 3,[3]BNode{leftleft,middle,right}
}
// -----------------------------------------------------------------------------
// Replace one child with multiple children
// -----------------------------------------------------------------------------
//
// The child at index idx in the old parent is replaced by the nodes in kids.
//
// Example:
//
// old parent:
//
// [A] [M] [Z]
//      |
//    child
//
// If child splits into:
//
// [M] and [T]
//
// the new parent becomes:
//
// [A] [M] [T] [Z]
//
// -----------------------------------------------------------------------------

func nodeReplaceKidN(
	tree *BTree,
	new BNode,
	old BNode,
	idx uint16,
	kids ...BNode,
) {
	// Number of new children replacing the old child.
	inc := uint16(len(kids))

	// We remove 1 old child and add inc new children.
	new.setHeader(BNODE_NODE, old.nkeys()+inc-1)

	// Copy everything before idx.
	nodeAppendRange(
		new,
		old,
		0,
		0,
		idx,
	)

	// Insert the new children.
	for i, node := range kids {
		nodeAppendKV(
			new,
			idx+uint16(i),
			tree.new(node),
			node.getKey(0),
			nil,
		)
	}

	// Copy everything after the old child.
	nodeAppendRange(
		new,
		old,
		idx+inc,
		idx+1,
		old.nkeys()-(idx+1),
	)
}
func treeInsert(tree *BTree, node BNode, key []byte, val []byte) BNode {
	// The new node can temporarily be larger than one page.
	// It will be split later using nodeSplit3().
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE))

	// Find the last key <= key.
	idx := nodeLookupLE(node, key)

	switch node.btype() {

	case BNODE_LEAF:
		// We reached a leaf.
		//
		// If the key already exists, update its value.
		// Otherwise, insert the new key after idx.

		if bytes.Equal(key, node.getKey(idx)) {
			leafUpdate(
				node,
				new,
				idx,
				key,
				val,
			)
		} else {
			leafInsert(
				node,
				new,
				idx+1,
				key,
				val,
			)
		}

	case BNODE_NODE:
		// Internal node.
		//
		// idx tells us which child subtree contains the key.

		kptr := node.getPointer(idx)

		// Recursively insert into that child.
		knode := treeInsert(
			tree,
			BNode(tree.get(kptr)),
			key,
			val,
		)

		// The child may now be too large.
		// Split it into 1, 2, or 3 nodes.
		nsplit, split := nodeSplit3(knode)

		// The old child is no longer needed because this
		// B+tree uses copy-on-write.
		tree.del(kptr)

		// Replace the old child with the newly created
		// 1, 2, or 3 children.
		nodeReplaceKidN(
			tree,
			new,
			node,
			idx,
			split[:nsplit]...,
		)

	default:
		panic("invalid BNode type")
	}

	return new
}

func treeGet(tree *BTree, key []byte) []byte {
	if tree.root == 0 {
		return nil
	}

	// Start at the root.
	node := BNode(tree.get(tree.root))

	for {
		idx := nodeLookupLE(node, key)

		switch node.btype() {

		case BNODE_LEAF:
			// We are at the leaf that should contain the key.

			if bytes.Equal(node.getKey(idx), key) {
				// Return the value associated with the key.
				return node.getVal(idx)
			}

			// Key does not exist.
			return nil

		case BNODE_NODE:
			// Internal node.
			//
			// idx tells us which child subtree to follow.
			ptr := node.getPointer(idx)

			node = BNode(tree.get(ptr))

		default:
			panic("invalid BNode type")
		}
	}
}

func (tree *BTree) Insert(key []byte, val []byte) {
	// -------------------------------------------------------------------------
	// Empty tree
	// -------------------------------------------------------------------------

	if tree.root == 0 {
		// Create the first leaf node.
		root := BNode(make([]byte, BTREE_PAGE_SIZE))

		// The first key is the sentinel.
		//
		// This guarantees that nodeLookupLE() always has
		// a valid key to return.
		root.setHeader(BNODE_LEAF, 1)

		nodeAppendKV(
			root,
			0,
			0,
			nil, // sentinel key
			nil, // sentinel value
		)

		tree.root = tree.new(root)
	}

	// -------------------------------------------------------------------------
	// Insert into the tree
	// -------------------------------------------------------------------------

	oldRoot := BNode(tree.get(tree.root))

	newRoot := treeInsert(
		tree,
		oldRoot,
		key,
		val,
	)

	// -------------------------------------------------------------------------
	// Split the root if necessary
	// -------------------------------------------------------------------------

	nsplit, split := nodeSplit3(newRoot)

	if nsplit == 1 {
		// Root still fits in one page.
		tree.del(tree.root)

		tree.root = tree.new(split[0])
		return
	}

	// Root was split.
	//
	// We need to create a new internal root whose children
	// are the split nodes.

	root := BNode(make([]byte, BTREE_PAGE_SIZE))

	root.setHeader(BNODE_NODE, nsplit)

	for i := uint16(0); i < nsplit; i++ {
		nodeAppendKV(
			root,
			i,
			tree.new(split[i]),
			split[i].getKey(0),
			nil,
		)
	}

	// The old root is no longer needed.
	tree.del(tree.root)

	// New internal root.
	tree.root = tree.new(root)
}
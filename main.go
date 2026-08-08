package main

import "fmt"

func main() {

	node := BNode(make([]byte, BTREE_PAGE_SIZE))

    node.setHeader(BNODE_LEAF, 2)

    nodeAppendKV(node, 0, 0, []byte("k1"), []byte("hi"))
    nodeAppendKV(node, 1, 0, []byte("k3"), []byte("hello"))

    fmt.Println(string(node.getKey(0)))
    fmt.Println(string(node.getVal(0)))

    fmt.Println(string(node.getKey(1)))
    fmt.Println(string(node.getVal(1)))

    fmt.Println("nbytes:", node.nbytes())
}
package trie

import (
	"fmt"
)

// Node provides implementation of the node at trie
type Node struct {
	val      rune
	term     bool
	meta     interface{}
	mask     uint64
	parent   *Node
	children map[rune]*Node
}

// Trie is a main structure
type Trie struct {
	root *Node
	size int
}

// ByKeys provides comparation of the keys on trie
type ByKeys []string

func (a ByKeys) Len() int           { return len(a) }
func (a ByKeys) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKeys) Less(i, j int) bool { return len(a[i]) < len(a[j]) }

const nul = 0x0

func newNode(parent *Node, val rune, m uint64, term bool) *Node {
	return &Node{
		val:      val,
		mask:     m,
		term:     term,
		parent:   parent,
		children: make(map[rune]*Node),
	}
}

// NewChild creates and returns a pointer to a new child for the node.
func (n *Node) NewChild(r rune, bitmask uint64, val rune, term bool) *Node {
	node := newNode(n, val, bitmask, term)
	n.children[r] = node
	return node
}

// RemoveChild provides removing of the child from node
func (n *Node) RemoveChild(r rune) {
	delete(n.children, r)

	n.recalculateMask()
	for parent := n.Parent(); parent != nil; parent = parent.Parent() {
		parent.recalculateMask()
	}
}

func (n *Node) recalculateMask() {
	n.mask = maskrune(n.Val())
	for k, c := range n.Children() {
		n.mask |= (maskrune(k) | c.Mask())
	}
}

// Parent returns the parent of this node.
func (n Node) Parent() *Node {
	return n.parent
}

// Meta information of this node.
func (n Node) Meta() interface{} {
	return n.meta
}

//Children of this node.
func (n Node) Children() map[rune]*Node {
	return n.children
}

// Val is a value of node
func (n Node) Val() rune {
	return n.val
}

// Mask returns a uint64 representing the current
// mask of this node.
func (n Node) Mask() uint64 {
	return n.mask
}

// NewTrie a new Trie with an initialized root Node.
func NewTrie() *Trie {
	node := newNode(nil, 0, 0, false)
	return &Trie{
		root: node,
		size: 0,
	}
}

// Root returns the root node for the Trie.
func (t *Trie) Root() *Node {
	return t.root
}

// Add the key to the Trie, including meta data.
func (t *Trie) Add(key string, meta interface{}) *Node {
	t.size++
	runes := []rune(key)
	node := t.addrune(t.Root(), runes, 0)
	node.meta = meta
	return node
}

// Find and returns node
func (t *Trie) Find(key string) (*Node, error) {
	node := t.nodeAtPath(key)
	node = node.Children()[nul]

	if node == nil || !node.term {
		err := fmt.Errorf("could not find key: %s in trie", key)
		return nil, err
	}

	return node, nil
}

// Remove a key from the trie, ensuring that
// all bitmasks up to root are appropriately recalculated.
func (t *Trie) Remove(key string) {
	var (
		i    int
		rs   = []rune(key)
		node = t.nodeAtPath(key)
	)

	t.size--
	for n := node.Parent(); n != nil; n = n.Parent() {
		i++
		if len(n.Children()) > 1 {
			r := rs[len(rs)-i]
			n.RemoveChild(r)
			break
		}
	}
}

// Keys returns all the keys currently stored in the trie.
func (t *Trie) Keys() []string {
	return t.PrefixSearch("")
}

// PrefixSearch a prefix search against the keys in the trie.
func (t Trie) PrefixSearch(pre string) []string {
	var keys []string

	node := t.nodeAtPath(pre)
	if node == nil {
		return keys
	}

	collect(node, []rune(pre), &keys)
	return keys
}

func (t Trie) nodeAtPath(pre string) *Node {
	runes := []rune(pre)
	return findNode(t.Root(), runes)
}

func findNode(node *Node, runes []rune) *Node {
	if node == nil {
		return nil
	}

	if len(runes) == 0 {
		return node
	}

	n, ok := node.Children()[runes[0]]
	if !ok {
		return nil
	}

	var nrunes []rune
	if len(runes) > 1 {
		nrunes = runes[1:]
	} else {
		nrunes = runes[0:0]
	}

	return findNode(n, nrunes)
}

func (t Trie) addrune(node *Node, runes []rune, i int) *Node {
	if len(runes) == 0 {
		return node.NewChild(0, 0, nul, true)
	}

	r := runes[0]
	c := node.Children()

	n, ok := c[r]
	bitmask := maskruneslice(runes)
	if !ok {
		n = node.NewChild(r, bitmask, r, false)
	}
	n.mask |= bitmask

	i++
	return t.addrune(n, runes[1:], i)
}

func maskruneslice(rs []rune) uint64 {
	var m uint64
	for _, r := range rs {
		m |= maskrune(r)
	}

	return m
}

func maskrune(r rune) uint64 {
	i := uint64(1)
	return i << (uint64(r) - 97)
}

func collect(node *Node, pre []rune, keys *[]string) {
	children := node.Children()
	for r, n := range children {
		if n.term {
			*keys = append(*keys, string(pre))
			continue
		}

		npre := append(pre, r)
		collect(n, npre, keys)
	}
}

func fuzzycollect(node *Node, partialmatch, partial []rune, keys *[]string) {
	partiallen := len(partial)

	if partiallen == 0 {
		collect(node, partialmatch, keys)
		return
	}

	m := maskruneslice(partial)
	children := node.Children()
	for v, n := range children {
		xor := n.Mask() ^ m
		if (xor & m) != 0 {
			continue
		}

		npartial := partial
		if v == partial[0] {
			if partiallen > 1 {
				npartial = partial[1:]
			} else {
				npartial = partial[0:0]
			}
		}

		fuzzycollect(n, append(partialmatch, v), npartial, keys)
	}
}

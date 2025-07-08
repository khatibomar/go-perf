package search

import (
	"strings"
)

// BSTNode represents a binary search tree node
type BSTNode struct {
	Value int
	Left  *BSTNode
	Right *BSTNode
}

// BST represents a binary search tree
type BST struct {
	Root *BSTNode
}

// NewBST creates a new binary search tree
func NewBST() *BST {
	return &BST{}
}

// Insert adds a value to the BST
func (bst *BST) Insert(value int) {
	bst.Root = bst.insertNode(bst.Root, value)
}

func (bst *BST) insertNode(node *BSTNode, value int) *BSTNode {
	if node == nil {
		return &BSTNode{Value: value}
	}

	if value < node.Value {
		node.Left = bst.insertNode(node.Left, value)
	} else if value > node.Value {
		node.Right = bst.insertNode(node.Right, value)
	}

	return node
}

// Search finds a value in the BST
func (bst *BST) Search(value int) bool {
	return bst.searchNode(bst.Root, value)
}

func (bst *BST) searchNode(node *BSTNode, value int) bool {
	if node == nil {
		return false
	}

	if value == node.Value {
		return true
	} else if value < node.Value {
		return bst.searchNode(node.Left, value)
	} else {
		return bst.searchNode(node.Right, value)
	}
}

// BTreeNode represents a B-Tree node
type BTreeNode struct {
	Keys     []int
	Children []*BTreeNode
	IsLeaf   bool
}

// BTree represents a B-Tree
type BTree struct {
	Root   *BTreeNode
	Degree int // Minimum degree
}

// NewBTree creates a new B-Tree
func NewBTree(degree int) *BTree {
	return &BTree{
		Root:   &BTreeNode{IsLeaf: true},
		Degree: degree,
	}
}

// Search finds a value in the B-Tree
func (bt *BTree) Search(value int) bool {
	return bt.searchNode(bt.Root, value)
}

func (bt *BTree) searchNode(node *BTreeNode, value int) bool {
	i := 0
	// Find the first key greater than or equal to value
	for i < len(node.Keys) && value > node.Keys[i] {
		i++
	}

	// If the found key is equal to value, return true
	if i < len(node.Keys) && value == node.Keys[i] {
		return true
	}

	// If this is a leaf node, value is not present
	if node.IsLeaf {
		return false
	}

	// Go to the appropriate child
	return bt.searchNode(node.Children[i], value)
}

// Insert adds a value to the B-Tree
func (bt *BTree) Insert(value int) {
	root := bt.Root

	// If root is full, create new root
	if len(root.Keys) == 2*bt.Degree-1 {
		newRoot := &BTreeNode{IsLeaf: false}
		newRoot.Children = append(newRoot.Children, root)
		bt.splitChild(newRoot, 0)
		bt.Root = newRoot
	}

	bt.insertNonFull(bt.Root, value)
}

func (bt *BTree) insertNonFull(node *BTreeNode, value int) {
	i := len(node.Keys) - 1

	if node.IsLeaf {
		// Insert into leaf
		node.Keys = append(node.Keys, 0)
		for i >= 0 && node.Keys[i] > value {
			node.Keys[i+1] = node.Keys[i]
			i--
		}
		node.Keys[i+1] = value
	} else {
		// Find child to insert into
		for i >= 0 && node.Keys[i] > value {
			i--
		}
		i++

		// Split child if full
		if len(node.Children[i].Keys) == 2*bt.Degree-1 {
			bt.splitChild(node, i)
			if node.Keys[i] < value {
				i++
			}
		}
		bt.insertNonFull(node.Children[i], value)
	}
}

func (bt *BTree) splitChild(parent *BTreeNode, index int) {
	fullChild := parent.Children[index]
	newChild := &BTreeNode{IsLeaf: fullChild.IsLeaf}

	// Move half the keys to new child
	midIndex := bt.Degree - 1
	
	// Store the median key before modifying the slice
	medianKey := fullChild.Keys[midIndex]
	
	newChild.Keys = make([]int, len(fullChild.Keys[midIndex+1:]))
	copy(newChild.Keys, fullChild.Keys[midIndex+1:])
	fullChild.Keys = fullChild.Keys[:midIndex]

	// Move children if not leaf
	if !fullChild.IsLeaf {
		newChild.Children = make([]*BTreeNode, len(fullChild.Children[midIndex+1:]))
		copy(newChild.Children, fullChild.Children[midIndex+1:])
		fullChild.Children = fullChild.Children[:midIndex+1]
	}

	// Insert new child into parent
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[index+2:], parent.Children[index+1:])
	parent.Children[index+1] = newChild

	// Move median key up to parent
	parent.Keys = append(parent.Keys, 0)
	copy(parent.Keys[index+1:], parent.Keys[index:])
	parent.Keys[index] = medianKey
}

// TrieNode represents a trie node
type TrieNode struct {
	Children map[rune]*TrieNode
	IsEnd    bool
	Value    interface{}
}

// Trie represents a trie (prefix tree)
type Trie struct {
	Root *TrieNode
}

// NewTrie creates a new trie
func NewTrie() *Trie {
	return &Trie{
		Root: &TrieNode{
			Children: make(map[rune]*TrieNode),
		},
	}
}

// Insert adds a word to the trie
func (t *Trie) Insert(word string, value interface{}) {
	node := t.Root
	for _, char := range word {
		if node.Children[char] == nil {
			node.Children[char] = &TrieNode{
				Children: make(map[rune]*TrieNode),
			}
		}
		node = node.Children[char]
	}
	node.IsEnd = true
	node.Value = value
}

// Search finds a word in the trie
func (t *Trie) Search(word string) (interface{}, bool) {
	node := t.Root
	for _, char := range word {
		if node.Children[char] == nil {
			return nil, false
		}
		node = node.Children[char]
	}
	return node.Value, node.IsEnd
}

// StartsWith checks if any word starts with the given prefix
func (t *Trie) StartsWith(prefix string) bool {
	node := t.Root
	for _, char := range prefix {
		if node.Children[char] == nil {
			return false
		}
		node = node.Children[char]
	}
	return true
}

// GetWordsWithPrefix returns all words with the given prefix
func (t *Trie) GetWordsWithPrefix(prefix string) []string {
	node := t.Root
	for _, char := range prefix {
		if node.Children[char] == nil {
			return nil
		}
		node = node.Children[char]
	}

	var words []string
	t.collectWords(node, prefix, &words)
	return words
}

func (t *Trie) collectWords(node *TrieNode, prefix string, words *[]string) {
	if node.IsEnd {
		*words = append(*words, prefix)
	}

	for char, child := range node.Children {
		t.collectWords(child, prefix+string(char), words)
	}
}

// CompressedTrie represents a compressed trie (radix tree)
type CompressedTrie struct {
	Root *CompressedTrieNode
}

type CompressedTrieNode struct {
	Edge     string
	Children map[byte]*CompressedTrieNode
	IsEnd    bool
	Value    interface{}
}

// NewCompressedTrie creates a new compressed trie
func NewCompressedTrie() *CompressedTrie {
	return &CompressedTrie{
		Root: &CompressedTrieNode{
			Children: make(map[byte]*CompressedTrieNode),
		},
	}
}

// Insert adds a word to the compressed trie
func (ct *CompressedTrie) Insert(word string, value interface{}) {
	node := ct.Root
	i := 0

	for i < len(word) {
		char := word[i]
		if child, exists := node.Children[char]; exists {
			// Find common prefix
			commonLen := 0
			for commonLen < len(child.Edge) && i+commonLen < len(word) && 
				child.Edge[commonLen] == word[i+commonLen] {
				commonLen++
			}

			if commonLen == len(child.Edge) {
				// Full edge match, continue
				node = child
				i += commonLen
			} else {
				// Partial match, split edge
				newChild := &CompressedTrieNode{
					Edge:     child.Edge[commonLen:],
					Children: child.Children,
					IsEnd:    child.IsEnd,
					Value:    child.Value,
				}

				child.Edge = child.Edge[:commonLen]
				child.Children = make(map[byte]*CompressedTrieNode)
				child.Children[newChild.Edge[0]] = newChild
				child.IsEnd = false
				child.Value = nil

				node = child
				i += commonLen
			}
		} else {
			// No matching child, create new one
			newChild := &CompressedTrieNode{
				Edge:     word[i:],
				Children: make(map[byte]*CompressedTrieNode),
				IsEnd:    true,
				Value:    value,
			}
			node.Children[char] = newChild
			return
		}
	}

	node.IsEnd = true
	node.Value = value
}

// Search finds a word in the compressed trie
func (ct *CompressedTrie) Search(word string) (interface{}, bool) {
	node := ct.Root
	i := 0

	for i < len(word) {
		char := word[i]
		if child, exists := node.Children[char]; exists {
			// Check if word matches edge
			if i+len(child.Edge) <= len(word) && 
				strings.HasPrefix(word[i:], child.Edge) {
				node = child
				i += len(child.Edge)
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	return node.Value, node.IsEnd
}

// SuffixTree represents a suffix tree
type SuffixTree struct {
	Root *SuffixNode
	Text string
}

type SuffixNode struct {
	Children map[byte]*SuffixNode
	Start    int
	End      *int
	SuffixLink *SuffixNode
	SuffixIndex int
}

// NewSuffixTree creates a new suffix tree
func NewSuffixTree(text string) *SuffixTree {
	st := &SuffixTree{
		Root: &SuffixNode{
			Children: make(map[byte]*SuffixNode),
			SuffixIndex: -1,
		},
		Text: text + "$", // Add terminator
	}
	st.build()
	return st
}

// build constructs the suffix tree by inserting all suffixes
func (st *SuffixTree) build() {
	n := len(st.Text)
	// Insert all suffixes
	for i := 0; i < n; i++ {
		st.insertSuffix(i)
	}
}

func (st *SuffixTree) insertSuffix(suffixStart int) {
	node := st.Root
	i := suffixStart

	for i < len(st.Text) {
		char := st.Text[i]
		if child, exists := node.Children[char]; exists {
			// Check how much of the edge matches
			j := 0
			for j < *child.End-child.Start+1 && i+j < len(st.Text) {
				if st.Text[child.Start+j] != st.Text[i+j] {
					break
				}
				j++
			}

			if j == *child.End-child.Start+1 {
				// Full edge match, continue
				node = child
				i += j
			} else {
				// Partial match, need to split
				oldEnd := *child.End
				newEnd := child.Start + j - 1
				child.End = &newEnd

				// Create new internal node
				newInternal := &SuffixNode{
					Children: make(map[byte]*SuffixNode),
					Start:    child.Start + j,
					End:      &oldEnd,
					SuffixIndex: -1,
				}
				child.Children[st.Text[child.Start+j]] = newInternal

				// Create new leaf for current suffix
				leafEnd := len(st.Text) - 1
				newLeaf := &SuffixNode{
					Children: make(map[byte]*SuffixNode),
					Start:    i + j,
					End:      &leafEnd,
					SuffixIndex: suffixStart,
				}
				child.Children[st.Text[i+j]] = newLeaf
				return
			}
		} else {
			// Create new leaf
			leafEnd := len(st.Text) - 1
			newLeaf := &SuffixNode{
				Children: make(map[byte]*SuffixNode),
				Start:    i,
				End:      &leafEnd,
				SuffixIndex: suffixStart,
			}
			node.Children[char] = newLeaf
			return
		}
	}
	// Mark as leaf if we've consumed the entire suffix
	node.SuffixIndex = suffixStart
}

// Search finds a pattern in the suffix tree
func (st *SuffixTree) Search(pattern string) []int {
	// Simple implementation: find all occurrences using string search
	var positions []int
	text := st.Text[:len(st.Text)-1] // Remove the $ terminator
	
	for i := 0; i <= len(text)-len(pattern); i++ {
		if text[i:i+len(pattern)] == pattern {
			positions = append(positions, i)
		}
	}
	
	return positions
}

func (st *SuffixTree) collectSuffixes(node *SuffixNode, positions *[]int) {
	// If this node represents a suffix (leaf or internal node with suffix)
	if node.SuffixIndex >= 0 {
		*positions = append(*positions, node.SuffixIndex)
	}

	// Recursively collect from children
	for _, child := range node.Children {
		st.collectSuffixes(child, positions)
	}
}
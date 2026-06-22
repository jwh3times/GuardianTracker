package collections

import (
	"sort"
	"strconv"

	"guardian-tracker/api-service/services/manifest"
)

// CollectionNode is one presentation node with rolled-up counts, as serialized to
// the frontend. Items holds the item hashes of this node's direct leaf collectibles
// (stripped on the lightweight response — see UserCollections.Lightweight).
type CollectionNode struct {
	Hash      string           `json:"hash"`
	Name      string           `json:"name"`
	Icon      string           `json:"icon"`
	Collected int              `json:"collected"`
	Total     int              `json:"total"`
	Children  []CollectionNode `json:"children,omitempty"`
	Items     []string         `json:"items,omitempty"`
}

// leafRef pairs a collectible (for the collected check, keyed by collectible hash)
// with its item hash (the frontend key for item detail + deep-link).
type leafRef struct {
	CollectibleHash uint32
	ItemHash        string
}

// structNode is the user-independent skeleton of one presentation node.
type structNode struct {
	Hash     uint32
	Name     string
	Icon     string
	Children []structNode
	Leaves   []leafRef
}

// TreeStructure is the cached, user-independent Collections forest plus the shared
// item-detail map. overlay turns it into counted CollectionNodes for one user.
type TreeStructure struct {
	Roots []structNode
	Items map[string]DestinyItem
}

// buildTreeStructure assembles the Collections forest from the full node map and
// collectible set. Roots are discovered (no hard-coded Collections root): a node is
// a root when it transitively contains a valid collectible and is not referenced as
// a child of any other collectible-bearing node.
func buildTreeStructure(nodes map[uint32]*manifest.PresentationNodeDef, collectibles []manifest.CollectibleWithItem) *TreeStructure {
	colByHash := make(map[uint32]manifest.CollectibleWithItem, len(collectibles))
	items := make(map[string]DestinyItem)
	for _, cwi := range collectibles {
		cwi := cwi
		if cwi.Item == nil || cwi.Item.DisplayProperties.Name == "" {
			continue
		}
		colByHash[cwi.Collectible.Hash] = cwi
		ih := strconv.FormatUint(uint64(cwi.Item.Hash), 10)
		if _, ok := items[ih]; !ok {
			items[ih] = toDestinyItem(&cwi)
		}
	}

	// contains[h] = does node h hold (transitively) a valid collectible?
	contains := make(map[uint32]bool)
	visiting := make(map[uint32]bool)
	var computeContains func(h uint32) bool
	computeContains = func(h uint32) bool {
		if v, ok := contains[h]; ok {
			return v
		}
		if visiting[h] {
			return false // cycle guard
		}
		n := nodes[h]
		if n == nil {
			contains[h] = false
			return false
		}
		visiting[h] = true
		res := false
		for _, c := range n.Children.Collectibles {
			if _, ok := colByHash[c.CollectibleHash]; ok {
				res = true
				break
			}
		}
		if !res {
			for _, c := range n.Children.PresentationNodes {
				if computeContains(c.PresentationNodeHash) {
					res = true
					break
				}
			}
		}
		visiting[h] = false
		contains[h] = res
		return res
	}

	// Mark which nodes are referenced as children, using a path set to avoid
	// marking cycle members as referenced from themselves.
	referenced := make(map[uint32]bool)
	var markReferenced func(h uint32, path map[uint32]bool)
	markReferenced = func(h uint32, path map[uint32]bool) {
		if !computeContains(h) || path[h] {
			return
		}
		n := nodes[h]
		if n == nil {
			return
		}
		path[h] = true
		for _, c := range n.Children.PresentationNodes {
			ch := c.PresentationNodeHash
			if computeContains(ch) && !path[ch] {
				referenced[ch] = true
				markReferenced(ch, path)
			}
		}
		delete(path, h)
	}
	for h := range nodes {
		markReferenced(h, map[uint32]bool{})
	}

	var rootHashes []uint32
	for h := range nodes {
		if computeContains(h) && !referenced[h] {
			rootHashes = append(rootHashes, h)
		}
	}
	// Fallback: if all collectible-bearing nodes are in a mutual cycle, every node
	// would be marked referenced and rootHashes would be empty. Break the tie by
	// selecting the node with the smallest hash from the collectible-bearing set.
	if len(rootHashes) == 0 {
		for h := range nodes {
			if computeContains(h) {
				rootHashes = append(rootHashes, h)
			}
		}
	}
	sort.Slice(rootHashes, func(i, j int) bool {
		return nodes[rootHashes[i]].DisplayProperties.Name < nodes[rootHashes[j]].DisplayProperties.Name
	})

	var build func(h uint32, path map[uint32]bool) structNode
	build = func(h uint32, path map[uint32]bool) structNode {
		n := nodes[h]
		sn := structNode{Hash: h, Name: n.DisplayProperties.Name, Icon: n.DisplayProperties.Icon}
		path[h] = true
		for _, c := range n.Children.PresentationNodes {
			ch := c.PresentationNodeHash
			if !computeContains(ch) || path[ch] {
				continue
			}
			sn.Children = append(sn.Children, build(ch, path))
		}
		for _, c := range n.Children.Collectibles {
			if cwi, ok := colByHash[c.CollectibleHash]; ok {
				sn.Leaves = append(sn.Leaves, leafRef{
					CollectibleHash: c.CollectibleHash,
					ItemHash:        strconv.FormatUint(uint64(cwi.Item.Hash), 10),
				})
			}
		}
		delete(path, h)
		return sn
	}

	ts := &TreeStructure{Items: items}
	for _, h := range rootHashes {
		ts.Roots = append(ts.Roots, build(h, map[uint32]bool{}))
	}
	return ts
}

// overlay produces counted CollectionNodes for a user's collected set (keyed by
// collectible hash).
func (ts *TreeStructure) overlay(collected map[uint32]bool) []CollectionNode {
	out := make([]CollectionNode, 0, len(ts.Roots))
	for _, r := range ts.Roots {
		out = append(out, overlayNode(r, collected))
	}
	return out
}

func overlayNode(n structNode, collected map[uint32]bool) CollectionNode {
	cn := CollectionNode{
		Hash: strconv.FormatUint(uint64(n.Hash), 10),
		Name: n.Name,
		Icon: n.Icon,
	}
	for _, c := range n.Children {
		child := overlayNode(c, collected)
		cn.Children = append(cn.Children, child)
		cn.Total += child.Total
		cn.Collected += child.Collected
	}
	for _, lf := range n.Leaves {
		cn.Total++
		if collected[lf.CollectibleHash] {
			cn.Collected++
		}
		cn.Items = append(cn.Items, lf.ItemHash)
	}
	return cn
}

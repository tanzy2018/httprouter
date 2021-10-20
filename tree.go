package httprouter

import (
	"sort"
)

type node struct {
	segament    string
	path        string
	isWildChild bool
	children0   [][]*node // 2D-slice promote the accuracy of the seeking for concret target node
	children1   []*node   // 1D-slice promote the accuracy of the seeking for fuzzy target node
	handle      Handle
	keyPair     []keyPair // save the params mode.
}

type keyPair struct {
	i   int // i the index that key in the path.
	key string
}

func findChild(children []*node, segament string) *node {
	// simple traverse
	if len(children) < 8 {
		for i := range children {
			if children[i].segament == segament {
				return children[i]
			}
		}
		return nil
	}
	// binary search
	i, j := 0, len(children)-1
	for i <= j {
		mid := i + (j-i)/2
		if children[mid].segament == segament {
			return children[mid]
		}
		if children[mid].segament > segament {
			j = mid - 1
			continue
		}
		if children[mid].segament < segament {
			i = mid + 1
			continue
		}
	}
	return nil
}

func (root *node) insertChild(path string, segments []string, handle Handle) {
	// TODO: complete the store logic.
	segment := segments[0]
	index := len(segments)
	var child *node
	type nodeType int
	const (
		old nodeType = iota
		newParam
		newConcret
	)
	nType := old
	if len(segment) == 1 && segment[0] == '*' {
		// handle the wildchild
		root.isWildChild = true
		if root.handle == nil {
			root.handle = handle
			root.path = path
		}
		root.keyPair = append(root.keyPair, resolveKeyPairFromPattern(path)...)
		sort.Slice(root.keyPair, func(i, j int) bool {
			return root.keyPair[i].i < root.keyPair[j].i
		})

		if root.children1 == nil {
			root.children1 = make([]*node, index+1)
		}
		if len(root.children1) < index+1 {
			root.children1 = append(root.children1, make([]*node, index+1-len(root.children1))...)
		}
		child = &node{
			isWildChild: true,
			path:        path,
			keyPair:     resolveKeyPairFromPattern(path),
			handle:      handle,
		}
		root.children1[index] = child
		return
	}

	if segment[0] == ':' {
		if root.children1 == nil {
			root.children1 = make([]*node, index+1)
		}
		if len(root.children1) < index+1 {
			root.children1 = append(root.children1, make([]*node, index+1-len(root.children1))...)
		}
		if root.children1[index] == nil {
			root.children1[index] = &node{}
			nType = newParam
		}
		child = root.children1[index]

	} else {
		if root.children0 == nil {
			root.children0 = make([][]*node, len(segments)+1)
		}
		if len(root.children0) < index+1 {
			root.children0 = append(root.children0,
				make([][]*node, index+1-len(root.children0))...)
			if root.children0[index] == nil {
				root.children0[index] = make([]*node, 0, 1)
			}
			child = findChild(root.children0[index], segment)
			if child == nil {
				root.children0[index] = append(root.children0[index], &node{segament: segment})
				child = root.children0[index][len(root.children0[index])-1]
				sort.Slice(root.children0[index], func(i, j int) bool {
					return root.children0[index][i].segament < root.children0[index][j].segament
				})
				nType = newConcret
			}
		}
	}

	if len(segments) == 1 && nType != old {
		child.handle = handle
		child.path = path
		child.keyPair = resolveKeyPairFromPattern(path)
	}
	if nType == newConcret {
		child.segament = segment
	}

	segments = segments[1:]
	if len(segments) > 0 {
		child.insertChild(path, segments, handle)
	}
}

func (root *node) resolvePath(path string) (handle Handle, ps Params) {
	return
}

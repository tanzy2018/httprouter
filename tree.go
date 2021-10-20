package httprouter

import (
	"sort"
	"strings"
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

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func (root *node) insertChild(path string, segments []string, handle Handle) {
	// TODO: complete the store logic.
	segment := segments[0]
	index := len(segments)
	var child *node
	type nodeType int
	const (
		old nodeType = iota
		newConcret
		newParam
		newWildchild
	)
	nType := old
	if segment[0] == ':' || segment[0] == '*' {
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
		if segment[0] == '*' {
			nType = newWildchild
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

	if nType == newWildchild {
		child.isWildChild = true
	}

	if len(segments) > 1 {
		child.insertChild(path, segments[1:], handle)
	}
}

func (root *node) resolvePath(path string, isWild bool) (handle Handle, ps Params) {
	if isWild {
		segaments := makeSegments(path, max(len(root.children0), len(root.children1)))
		return root.getValue1(path, segaments)
	} else {
		segaments := strings.Split(strings.ToLower(path), "/")
		return root.getValue(path, segaments)
	}
}

func (root *node) getValue(path string, segaments []string) (handle Handle, ps Params) {
	if len(segaments) == 0 {
		if root.handle == nil {
			return
		}
		handle = root.handle
		if len(root.keyPair) > 0 {
			ps = resolveParamsFromPath(path, root.keyPair, root.isWildChild)
		}
		return
	}
	// try finding handle in the children0
	if len(segaments) <= len(root.children0) {
		child := findChild(root.children0[len(segaments)], segaments[0])
		if child != nil {
			handle, ps = child.getValue(path, segaments[1:])
		}
	}

	// try finding handle in the children1
	if handle == nil && len(segaments) <= len(root.children1) {
		if child := root.children1[len(segaments)]; child != nil {
			handle, ps = child.getValue(path, segaments[1:])
		}
	}
	return
}

func (root *node) getValue1(path string, segaments []string) (handle Handle, ps Params) {
	if len(segaments) == 0 {
		if root.handle == nil {
			return
		}
		handle = root.handle
		if len(root.keyPair) > 0 {
			ps = resolveParamsFromPath(path, root.keyPair, root.isWildChild)
		}
		return
	}
	// try getValue
	for len(segaments) >= 0 {
		handle, ps = root.getValue(path, segaments)
		if handle != nil {
			break
		}
		segaments = segaments[:len(segaments)-1]
	}

	return
}

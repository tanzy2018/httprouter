package httprouter

type nodeType int

const (
	static      nodeType = iota
	fileService          // file server node
)

type node struct {
	segament  string
	path      string
	wildChild bool
	children0 [][]*node // 2D-slice promote the accuracy of the seeking for concret target node
	children1 [][]*node // 2D-slice promote the accuracy of the seeking for fuzzy target node
	handle    Handle
	nodeType  nodeType
	keyPair   []keyPair // save the params mode.
}

type keyPair struct {
	i   int // i the index that key in the path.
	key string
}

func (root *node) addNode(path string, paths []string, isFileServe bool) {}

func (root *node) resolvePath(path string) (handle Handle, ps Params) {
	return
}

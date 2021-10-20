package httprouter

import (
	"fmt"
	"net/http"
	filePath "path"
	"sort"
	"strings"
)

type methodType int

const (
	methodCount = 7
)

var methods = [methodCount]string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodHead,
	http.MethodPatch,
	http.MethodOptions,
}

func methodString2MethodType(method string) methodType {
	for i := 0; i < methodCount; i++ {
		if methods[i] == method {
			return methodType(i)
		}
	}
	return -1
}

func assertMethod(method string) {
	for i := 0; i < methodCount; i++ {
		if methods[i] == method {
			return
		}
	}
	panic("un resolved method type: " + method)
}

type pathsBitsMap struct {
	_map [methodCount]map[string]string
}

func assertPath(mt methodType, path string) {
	if len(path) == 0 {
		panic("path must not be empty string")
	}
	if path[0] != '/' {
		panic("path must start with '/'")
	}
	bitsMap := pathsMap._map[mt]
	if bitsMap != nil && bitsMap[unifyPattern(path)] != "" {
		panic(fmt.Sprintf("path0 [%s] and path1 [%s] conflict for the same pattern.", path, bitsMap[path]))
	}
}

func assertPathPrefixWithFileServe(path string, nodes []*node) {
	for _, n := range nodes {
		if len(n.segament) <= len(path) && n.segament == path[:len(n.segament)] {
			panic(fmt.Sprintf("the prefix of path0 [%s] is conflict with path1 [%s] for the same pattern.", path, n.path))
		}
	}
}

func insertPathPattern(mt methodType, path string) {
	if pathsMap._map[mt] == nil {
		pathsMap._map[mt] = make(map[string]string)
	}
	pathsMap._map[mt][unifyPattern(path)] = path
}

func assertSegaments(segaments []string, max int) {
	if len(segaments) > max {
		panic(fmt.Sprintf("the segament is overfollow, max segaments is %d ", max))
	}
	for i := range segaments {
		if segaments[i][0] == '*' && i != len(segaments)-1 {
			panic("/*path mode can only set at the end")
		}
	}
}

func assertSegamentsForFileServe(segaments []string) {
	for i := range segaments {
		if segaments[i][0] == '*' && i != len(segaments)-1 {
			panic("/:path mode can not set in the static file server")
		}
	}
}

var pathsMap *pathsBitsMap

type Handle func(http.ResponseWriter, *http.Request, Params)

type Param struct {
	Key   string
	Value string
}

type Params []Param

func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

var MatchedRoutePathParam = "$matchedRoutePath"

func (ps Params) MatchedRoutePath() string {
	return ps.ByName(MatchedRoutePathParam)
}

type Router struct {
	trees                  [methodCount]*node
	fileServes             []*node
	isFileServe            bool
	PanicHandler           func(http.ResponseWriter, *http.Request, interface{})
	NotFound               http.HandlerFunc
	MethodNotAllowed       http.HandlerFunc
	HandleMethodNotAllowed bool
	isContainsFileService  bool
	HandleOPTIONS          bool
	GlobalOPTIONS          http.Handler
	// Cached value of global (*) allowed methods
	globalAllowed string
	MaxSegaments  int
}

func New() *Router {
	return &Router{
		HandleMethodNotAllowed: true,
		isContainsFileService:  true,
		HandleOPTIONS:          true,
		MaxSegaments:           16,
	}
}

func (r *Router) GET(path string, handle Handle) {
	if r.isFileServe {

	}
	r.Handle(http.MethodGet, path, handle)
}

func (r *Router) HEAD(path string, handle Handle) {
	r.Handle(http.MethodHead, path, handle)
}

func (r *Router) OPTIONS(path string, handle Handle) {
	r.Handle(http.MethodOptions, path, handle)
}

func (r *Router) POST(path string, handle Handle) {
	r.Handle(http.MethodPost, path, handle)
}

func (r *Router) PUT(path string, handle Handle) {
	r.Handle(http.MethodPut, path, handle)
}

func (r *Router) PATCH(path string, handle Handle) {
	r.Handle(http.MethodPatch, path, handle)
}

func (r *Router) DELETE(path string, handle Handle) {
	r.Handle(http.MethodDelete, path, handle)
}

func (r *Router) Handle(method string, path string, handle Handle) {
	method = strings.ToUpper(method)
	assertMethod(method)
	mT := methodString2MethodType(method)
	r.handle(mT, path, handle)
}

func (r *Router) handle(method methodType, path string, handle Handle) {
	path = filePath.Clean(path)
	assertPath(method, path)
	if r.isFileServe && method == methodString2MethodType(http.MethodGet) {
		assertPathPrefixWithFileServe(path, r.fileServes)
	}

	root := r.trees[method]
	if root == nil {
		r.trees[method] = &node{
			segament: "/",
		}
		root = r.trees[method]
		if path == "/" {
			root.path = "/"
			root.handle = handle
			return
		}
	}
	segaments := strings.Split(strings.ToLower(path), "/")
	assertSegaments(segaments, r.MaxSegaments)
	insertPathPattern(method, path)
	root.insertChild(path, segaments, handle)
}

func (r *Router) handleFileServe(method methodType, path string, handle Handle) {
	path = filePath.Clean(path)
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}
	assertPath(method, path)
	segaments := strings.Split(strings.ToLower(path), "/")
	assertSegaments(segaments, r.MaxSegaments)
	assertSegamentsForFileServe(segaments)
	insertPathPattern(method, path[:len(path)-10])
	r.isFileServe = true
	fnode := &node{
		segament:    path[:len(path)-10],
		path:        path,
		handle:      handle,
		isWildChild: true,
		keyPair:     []keyPair{{i: len(segaments) - 1, key: "filepath"}},
	}
	r.fileServes = append(r.fileServes, fnode)
	if len(r.fileServes) > 1 {
		sort.Slice(r.fileServes, func(i, j int) bool {
			return r.fileServes[i].path > r.fileServes[j].path
		})
	}
}

func (r *Router) Handler(method, path string, handler http.Handler) {
	r.Handle(method, path,
		func(w http.ResponseWriter, req *http.Request, p Params) {
			handler.ServeHTTP(w, req)
		},
	)
}

func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.Handler(method, path, handler)
}

func (r *Router) ServeFiles(path string, root http.FileSystem) {
	fileServer := http.FileServer(root)
	r.handleFileServe(methodString2MethodType(http.MethodGet), path, func(w http.ResponseWriter, req *http.Request, ps Params) {
		req.URL.Path = ps.ByName("filepath")
		fileServer.ServeHTTP(w, req)
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}

	path := req.URL.Path
	if r.isFileServe && req.Method == http.MethodGet {
		// handle the static file server
		for _, root := range r.fileServes {
			if root.path == path[:len(root.path)] {
				root.handle(w, req, resolveParamsFromPath(path, root.keyPair, root.isWildChild))
				return
			}
		}
	}
	if root := r.trees[methodString2MethodType(req.Method)]; root != nil {
		if handle, ps := root.resolvePath(path); handle != nil {
			if ps != nil {
				handle(w, req, ps)
			} else {
				handle(w, req, nil)
			}
			return
		}
	}

	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		if allow := r.allowed(path, http.MethodOptions); allow != "" {
			w.Header().Set("Allow", allow)
			if r.GlobalOPTIONS != nil {
				r.GlobalOPTIONS.ServeHTTP(w, req)
			}
			return
		}
	} else if r.HandleMethodNotAllowed { // Handle 405
		if allow := r.allowed(path, req.Method); allow != "" {
			w.Header().Set("Allow", allow)
			if r.MethodNotAllowed != nil {
				r.MethodNotAllowed.ServeHTTP(w, req)
			} else {
				http.Error(w,
					http.StatusText(http.StatusMethodNotAllowed),
					http.StatusMethodNotAllowed,
				)
			}
			return
		}
	}

	if r.NotFound != nil {
		r.NotFound.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (r *Router) Lookup(method, path string) (Handle, Params, bool) {
	return nil, nil, false
}

func (r *Router) recv(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(w, req, rcv)
	}
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	return
}

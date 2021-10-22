package httprouter

import (
	"fmt"
	"net/http"
	filePath "path"
	"sort"
	"strings"
	"sync"
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
	patterns [methodCount]map[string]string
	wilds    [methodCount]bool
}

func (pb *pathsBitsMap) insertPathPattern(mt methodType, path string) {
	if pb.patterns[mt] == nil {
		pb.patterns[mt] = make(map[string]string)
	}
	pb.patterns[mt][unifyPattern(path)] = path
}

func (pb *pathsBitsMap) searchPattern(mt methodType, path string) (existedPath string) {
	bitsMap := pb.patterns[mt]
	if bitsMap != nil {
		return bitsMap[unifyPattern(path)]
	}
	return ""
}

func (pb *pathsBitsMap) setWildsBit(mt methodType, isWild bool) {
	if !pb.wilds[mt] {
		pb.wilds[mt] = isWild
	}
}

func (pb *pathsBitsMap) getWildsBitByMethodType(mt methodType) bool {
	return pb.wilds[mt]
}

func assertPath(mt methodType, path string) {
	if len(path) == 0 {
		panic("path must not be empty string")
	}
	if path[0] != '/' {
		panic("path must start with '/'")
	}
	if existedPath := pathsMap.searchPattern(mt, path); existedPath != "" {
		panic(fmt.Sprintf("inserting path [%s] conficts with the exsited path [%s] for the same pattern.", path, existedPath))
	}
}

func assertPathPrefixWithFileServe(path string, nodes []*node) {
	for _, n := range nodes {
		if len(n.segament) <= len(path) && n.segament == path[:len(n.segament)] {
			panic(fmt.Sprintf("the prefix of path0 [%s] is conflict with path1 [%s] for the same pattern.", path, n.path))
		}
	}
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

var pathsMap = new(pathsBitsMap)

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

type paramsPool struct {
	pools []sync.Pool
}

func (pp *paramsPool) update(count int) {
	if len(pp.pools) < count {
		pp.pools = append(pp.pools, sync.Pool{})
	}
	pp.update(count - 1)
}

func (pp *paramsPool) put(ps Params, index int) {
	pp.pools[index].Put(&ps)
}

func (pp *paramsPool) get(index int) Params {
	ps := pp.pools[index].Get()
	if ps == nil {
		return nil
	}
	return *(ps.(*Params))
}

var paramsPools = new(paramsPool)
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
	globalAllowed     string
	MaxSegamentsLimit int
}

func New() *Router {
	return &Router{
		HandleMethodNotAllowed: true,
		isContainsFileService:  true,
		HandleOPTIONS:          true,
		MaxSegamentsLimit:      16,
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
	assertSegaments(segaments, r.MaxSegamentsLimit)
	if segaments[len(segaments)-1][0] == '*' {
		subPath := path[:len(path)-len(segaments[len(segaments)-1])-1]
		pathsMap.insertPathPattern(method, strings.ToLower(subPath))
		pathsMap.setWildsBit(method, true)
		root.insertChild(subPath, segaments[:len(segaments)-1], handle)
	}
	pathsMap.insertPathPattern(method, strings.ToLower(path))
	root.insertChild(path, segaments, handle)
}

func (r *Router) handleFileServe(method methodType, path string, handle Handle) {
	path = filePath.Clean(path)
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}
	assertPath(method, path)
	segaments := strings.Split(strings.ToLower(path), "/")
	assertSegaments(segaments, r.MaxSegamentsLimit)
	assertSegamentsForFileServe(segaments)
	pathsMap.insertPathPattern(method, strings.ToLower(path[:len(path)-10]))
	r.isFileServe = true
	fnode := &node{
		segament:    path[:len(path)-10],
		path:        path,
		handle:      handle,
		isWildChild: true,
		keyPair:     []keyPair{{i: len(segaments) - 1, key: "filepath"}},
	}
	paramsPools.update(1)
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

	path := filePath.Clean(req.URL.Path)
	if r.isFileServe && req.Method == http.MethodGet {
		// handle the static file server
		_path := strings.ToLower(path)
		for _, root := range r.fileServes {
			if root.path == _path[:len(root.path)] {
				ps := resolveParamsFromPath(path, root.keyPair, root.isWildChild)
				root.handle(w, req, ps)
				paramsPools.put(ps, 0)
				return
			}
		}
	}
	mt := methodString2MethodType(req.Method)
	if root := r.trees[mt]; root != nil {
		if handle, ps := root.resolvePath(path, pathsMap.getWildsBitByMethodType(mt)); handle != nil {
			if ps != nil {
				handle(w, req, ps)
				paramsPools.put(ps, len(ps)-1)
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

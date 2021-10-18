package httprouter

import (
	"fmt"
	"net/http"
	filePath "path"
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

type pathsBitsMap struct {
	_map [methodCount]map[string]string
}

func assertPath(mt methodType, path string) {
	bitsMap := pathsMap._map[mt]
	if bitsMap != nil && bitsMap[unifyPattern(path)] != "" {
		panic(fmt.Sprintf("path0 [%s] and path1 [%s] conflict for the same pattern.", path, bitsMap[path]))
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
	PanicHandler           func(http.ResponseWriter, *http.Request, interface{})
	NotFound               http.HandlerFunc
	MethodNotAllowed       http.HandlerFunc
	HandleMethodNotAllowed bool
	isContainsFileService  bool
	HandleOPTIONS          bool
	GlobalOPTIONS          http.Handler

	// Cached value of global (*) allowed methods
	globalAllowed string
}

func New() *Router {
	return &Router{
		HandleMethodNotAllowed: true,
		isContainsFileService:  true,
		HandleOPTIONS:          true,
	}
}

func (r *Router) GET(path string, handle Handle) {
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
	mT := methodString2MethodType(strings.ToUpper(method))
	path = filePath.Clean(path)
	r.handle(mT, path, false, handle)
}

func (r *Router) handle(method methodType, path string, isFileServe bool, handle Handle) {
	assertPath(method, path)
	// TODO: addNode
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
	path = filePath.Clean(path)
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)
	r.handle(methodString2MethodType("GET"), path, true, func(w http.ResponseWriter, req *http.Request, ps Params) {
		req.URL.Path = ps.ByName("filepath")
		fileServer.ServeHTTP(w, req)
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}

	path := req.URL.Path

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

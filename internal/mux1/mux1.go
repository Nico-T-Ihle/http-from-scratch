package mux1

import (
	"net/http"
	"path"
	"strings"
	"sync"
)

// TODO: the differences with post get delete etc.

type ServeMux struct {
	mux    sync.RWMutex            // so that only on can use it at the time if i understood it correctly
	routes map[string]http.Handler // the path
}

func NewServerMux() *ServeMux {
	return &ServeMux{
		routes: make(map[string]http.Handler),
	}
}

func (m *ServeMux) Handle(pattern string, handler http.Handler) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if pattern == "" {
		panic("http: invalid pattern: pattern cannot be empty")
	}

	if pattern[0] != '/' {
		panic("http: invalid pattern: must start with a '/'")
	}

	if handler == nil {
		panic("http: invalid pattern: must only have a single handler")
	}

	m.routes[pattern] = handler
}

func (m *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Handle(pattern, http.HandlerFunc(handler))
}

// match sucht den passenden Handler für einen Pfad
func (m *ServeMux) match(path string) http.Handler {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if handler, exists := m.routes[path]; exists {
		println("handler", handler)
		return handler
	}
	var mostSpecificHandler http.Handler
	maxLen := 0
	// in case there is a tree path like /hello/../.. so if that exists don't just shows 404
	for pattern, handler := range m.routes {
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) {
			if len(pattern) > maxLen {
				println("pattern too long", maxLen, pattern)
				maxLen = len(pattern)
				mostSpecificHandler = handler
			}
		}
	}
	println("mostSpecificHandler", mostSpecificHandler)
	return mostSpecificHandler
}

// start point of every request
func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// should then redirect it clean it
	cleanPath := path.Clean(r.URL.Path)

	handler := m.match(cleanPath)

	if handler == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 page not found\n"))
		return
	}

	handler.ServeHTTP(w, r)
}

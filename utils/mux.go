package utils

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
)

type muxEntry struct {
	pattern string
	handler interface{}
}

// Mux is an command multiplexer. It matches the URL of each incoming command
// against a list of registered patterns and calls the handler for the pattern
// that most closely matches the URL.
type Mux struct {
	mtx   sync.RWMutex
	table map[string]muxEntry
	array []muxEntry
}

// Init this class
func (me *Mux) Init() *Mux {
	me.table = make(map[string]muxEntry)
	me.array = make([]muxEntry, 0)
	return me
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (me *Mux) Handle(pattern string, handler interface{}) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	if pattern == "" {
		panic("invalid pattern")
	}
	if handler == nil {
		panic("nil handler")
	}
	if _, ok := me.table[pattern]; ok {
		panic(fmt.Sprintf("multiple registrations for %s" + pattern))
	}

	e := muxEntry{pattern: pattern, handler: handler}
	me.table[pattern] = e

	if pattern[len(pattern)-1] == '/' {
		me.array = appendOnSort(me.array, e)
	}
}

// Handler returns the handler to use for the given path of nc.
// It always returns a non-nil handler.
func (me *Mux) Handler(u *url.URL) (interface{}, string) {
	// All other requests have any port stripped and path cleaned before passing to mux.handler
	host := stripHostPort(u.Host)
	path := clean(u.Path)

	me.mtx.RLock()
	defer me.mtx.RUnlock()

	// If the given path is /tree and its handler is not registered, redirect for /tree/
	if ok := me.shouldRedirect(host, path); ok {
		path += "/"
	}

	return me.match(path)
}

// shouldRedirect reports whether the given path and host should be redirected to
// path+"/". This should happen if a handler is registered for path+"/" but
// not path -- see comments at Mux.
func (me *Mux) shouldRedirect(host, path string) bool {
	arr := []string{path, host + path}

	for _, pattern := range arr {
		if _, ok := me.table[pattern]; ok {
			return false
		}
	}

	n := len(path)
	if n == 0 {
		return false
	}

	for _, pattern := range arr {
		if _, ok := me.table[pattern+"/"]; ok {
			return path[n-1] != '/'
		}
	}

	return false
}

// Find a handler on a handler map given a path string.
// Most-specific (longest) pattern wins.
func (me *Mux) match(path string) (interface{}, string) {
	v, ok := me.table[path]
	if ok {
		return v.handler, v.pattern
	}

	// Check for longest valid match. me.array contains all patterns
	// that end in / sorted from longest to shortest.
	for _, e := range me.array {
		if strings.HasPrefix(path, e.pattern) {
			return e.handler, e.pattern
		}
	}

	return nil, ""
}

func appendOnSort(arr []muxEntry, e muxEntry) []muxEntry {
	n := len(arr)

	i := sort.Search(n, func(i int) bool {
		return len(arr[i].pattern) < len(e.pattern)
	})

	if i == n {
		return append(arr, e)
	}

	// We now know that i points at where we want to insert
	arr = append(arr, muxEntry{}) // Try to grow the slice in place, any entry works
	copy(arr[i+1:], arr[i:])      // Move shorter entries down
	arr[i] = e

	return arr
}

// stripHostPort returns h without any trailing ":<port>".
func stripHostPort(host string) string {
	// If no port on host, return unchanged
	if strings.IndexByte(host, ':') == -1 {
		return host
	}

	h, _, err := net.SplitHostPort(host)
	if err != nil {
		return host // on error, return unchanged
	}

	return h
}

// clean returns the canonical path for p, eliminating . and .. elements.
func clean(p string) string {
	if p == "" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}

	np := path.Clean(p)

	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		// Fast path for common case of p being the string we want:
		if len(p) == len(np)+1 && strings.HasPrefix(p, np) {
			np = p
		} else {
			np += "/"
		}
	}

	return np
}

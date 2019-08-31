package xo

import (
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

// APIMux is an API request multiplexer.
// It matches the APIMux.key of each incoming request
// on the APIMux.path against a list of registered API names
// and calls the handler for the API name that matches
// the requested APIMux.key value.
//
// If not one of the registered API names does not match
// the requested API name, APIMux calls the view handler
// that serves the APIMux.pub pulic folder path.
type APIMux struct {
	mu   sync.RWMutex
	pub  string // public folder path
	path string // API URL path
	key  string // API name query key
	m    map[string]http.Handler
	v    http.Handler // view handler
}

// Mux is the default APIMux used by Serve.
var Mux = &APIMux{
	pub:  "./public",
	path: "/api",
	key:  "name",
	v:    http.NotFoundHandler(),
}

// SetPublic sets the public folder path.
func (mux *APIMux) SetPublic(folderPath string) {
	if fi, err := os.Stat(folderPath); os.IsNotExist(err) || !fi.IsDir() {
		panic("xo: given pulic folder path is not valid: " + folderPath)
	}
	mux.pub = folderPath
}

// SetPath sets the API URL path.
func (mux *APIMux) SetPath(path string) {
	m, err := regexp.MatchString(`^(\/[\w-]+)+$`, path)
	if err != nil {
		panic("xo: regexp error on path validation...\n" + err.Error())
	}
	if !m {
		panic("xo: given API URL path is invalid: " + path)
	}
	mux.path = path
}

// SetKey sets the API name query key.
func (mux *APIMux) SetKey(key string) {
	m, err := regexp.MatchString(`^[\w-]+$`, key)
	if err != nil {
		panic("xo: regexp error on API name query key validation...\n" + err.Error())
	}
	if !m {
		panic("xo: given API name query key is invalid: " + key)
	}
	mux.key = key
}

// Handler returns the handler to use for the given request,
// consulting r.URL.Path, r.URL.Query.
//
// If the URL path is different from APIMux.path or there is
// no registered handler that applies to the request,
// Handler returns a nil handler.
func (mux *APIMux) Handler(r *http.Request) (h http.Handler) {
	if r.URL.Path != mux.path {
		return nil
	}

	mux.mu.RLock()
	defer mux.mu.RUnlock()

	h, ok := mux.m[strings.ToLower(r.URL.Query().Get(mux.key))]
	if ok {
		return
	}

	return nil
}

// Handle registers the handler for the given API name or
// registers the handler as a new view handler in case
// that APIName is an empty string.
// If a handler already exists for the given API name,
// Handle panics.
func (mux *APIMux) Handle(APIName string, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if handler == nil {
		panic("xo: nil handler")
	}
	if APIName == "" {
		mux.v = handler
	}
	if _, exist := mux.m[APIName]; exist {
		panic("xo: multiple registrations for " + APIName)
	}

	if mux.m == nil {
		mux.m = make(map[string]http.Handler)
	}
	mux.m[strings.ToLower(APIName)] = handler
}

// HandleFunc registers the handler function for the given
// API name or registers the handler function as a new
// view handler if APIName is an empty string.
func (mux *APIMux) HandleFunc(APIName string, handler func(http.ResponseWriter, *http.Request)) {
	if handler == nil {
		panic("xo: nil handler")
	}
	mux.Handle(APIName, http.HandlerFunc(handler))
}

// The serveView method dispatches the request to the
// view handler.
func (mux *APIMux) serveView(w http.ResponseWriter, r *http.Request) {
	requestedPath := path.Join(mux.pub, r.URL.Path)

	// serve files from the public path
	if fi, err := os.Stat(requestedPath); err == nil && !fi.IsDir() {
		http.FileServer(http.Dir(mux.pub)).ServeHTTP(w, r)
		return
	}

	// remove trailing slash
	if l := len(r.URL.Path) - 1; l > 0 && strings.HasSuffix(r.URL.Path, "/") {
		uri := r.URL.Path[:l]
		if r.URL.RawQuery != "" {
			uri += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, uri, http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	mux.v.ServeHTTP(w, r)
}

// ServeHTTP dispatches the request to the handler whose
// API name matches the requested APIMux.key value
// or to the view handler in case that there is no match.
func (mux *APIMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := mux.Handler(r)

	if h == nil {
		mux.serveView(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	h.ServeHTTP(w, r)
}

// Handle registers the handler for the given API name in
// the Mux.
func Handle(apiName string, handler http.Handler) {
	Mux.Handle(apiName, handler)
}

// HandleFunc registers the handler function for the given
// API name in the Mux.
func HandleFunc(apiName string, handler func(http.ResponseWriter, *http.Request)) {
	Mux.HandleFunc(apiName, handler)
}

// Serve registers the Mux handler to handle requests
// and then calls http.ListenAndServe with address addr and
// handler http.DefaultServeMux.
//
// If the given public folder does not exist, Serve panics.
//
// Serve always returns a non-nil error.
func Serve(addr string) error {
	http.Handle("/", Mux)
	return http.ListenAndServe(addr, http.DefaultServeMux)
}

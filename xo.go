package xo

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
)

type Route struct {
	value   reflect.Value
	methods map[string]reflect.Value
}

func Run(router any) {
	routerType := reflect.TypeOf(router)
	routerValue := reflect.ValueOf(router)
	routes := make(map[string]Route)

	// Routes
	for i := 0; i < routerType.NumField(); i++ {
		routeType := routerType.Field(i).Type
		routeValue := routerValue.Field(i)
		routeName := strings.ToLower(routeType.Name())
		routes[routeName] = Route{
			value:   routeValue,
			methods: make(map[string]reflect.Value),
		}

		// Actions
		for j := 0; j < routeType.NumMethod(); j++ {
			actionType := routeType.Method(j)
			actionName := strings.ToLower(actionType.Name)
			routes[routeName].methods[actionName] = actionType.Func
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestedPath := path.Join("./public", r.URL.Path)

		// Serve files
		if fi, err := os.Stat(requestedPath); err == nil && !fi.IsDir() {
			http.FileServer(http.Dir("./public")).ServeHTTP(w, r)
			return
		}

		path := r.URL.Path[1:]
		l := len(path)

		// Remove trailing slash
		if l > 0 && strings.HasSuffix(path, "/") {
			uri := r.URL.Path[:l]
			if r.URL.RawQuery != "" {
				uri += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, uri, http.StatusMovedPermanently)
			return
		}

		// Parse route name
		if l == 0 {
			fmt.Fprint(w, "Home page")
			return
		}

		parts := strings.Split(r.URL.Path[1:], "/")
		count := len(parts)
		routeName := strings.ToLower(parts[0])
		methodName := "index"

		if count > 1 {
			methodName = strings.ToLower(parts[1])
		}

		if route, ok := routes[routeName]; ok {
			if method, ok := route.methods[methodName]; ok {
				res := method.Call([]reflect.Value{
					route.value,
				})[0].String()
				fmt.Fprint(w, res)
			} else {
				fmt.Fprint(w, "Method not found")
			}
		} else {
			fmt.Fprint(w, "Route not found")
		}
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}

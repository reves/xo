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

func getRoutes(router any) map[string]Route {
	routes := make(map[string]Route)
	routerType := reflect.TypeOf(router)
	routerValue := reflect.ValueOf(router)

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

	return routes
}

func Run(router any, view http.HandlerFunc) {
	routes := getRoutes(router)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Serve files
		requestedPath := path.Join("./public", r.URL.Path)
		if fi, err := os.Stat(requestedPath); err == nil && !fi.IsDir() {
			http.FileServer(http.Dir("./public")).ServeHTTP(w, r)
			return
		}

		// Get URL path
		path := strings.ToLower(r.URL.Path[1:])
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

		// View
		if l == 0 || !strings.HasPrefix(path, "api/") {
			view(w, r)
			return
		}

		// Api
		path = strings.Replace(path, "api/", "", 1)
		parts := strings.Split(path, "/")
		count := len(parts)
		routeName := parts[0]

		methodName := "index"
		if count > 1 {
			methodName = parts[1]
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

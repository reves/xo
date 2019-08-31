# xo

Application example:

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/reves/xo"
)

func width(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"width": 30}`)
}

func front(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `Hi there, %s!`, r.URL.Path[1:])
}

func main() {
	xo.HandleFunc("width", width) // http://127.0.0.1:8080/api?name=width
	xo.HandleFunc("", front) // http://127.0.0.1:8080/*
	log.Fatal(xo.Serve("127.0.0.1:8080"))
}
```
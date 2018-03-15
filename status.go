package main

import (
	"fmt"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1> Welcome on the status page</h1>")
}

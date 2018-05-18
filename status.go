package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got http event for /status")
	t, _ := template.ParseFiles("templates/status.html")
	t.Execute(w, nil)
}

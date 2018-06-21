package main

import (
	"html/template"
	"log"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Got http event for /status")
	t, _ := template.ParseFiles("templates/status.html")
	t.Execute(w, nil)
}

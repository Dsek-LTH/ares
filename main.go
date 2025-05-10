package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed views/*
var views embed.FS
var t = template.Must(template.ParseFS(views, "views/*"))

func main() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, "index.html", nil); err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = server.ListenAndServe()
}

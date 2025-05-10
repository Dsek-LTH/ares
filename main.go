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
	router.HandleFunc("/index", index)
	router.HandleFunc("/admin", admin)
	router.HandleFunc("/sign-up", signup)
	router.HandleFunc("/leaderboard", leaderboard)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = server.ListenAndServe()
}

func index(w http.ResponseWriter, r *http.Request) {
	if err := t.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
	if 0 == 1 {
		if err := t.ExecuteTemplate(w, "userpage.html", nil); err != nil {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
	}
}
func admin(w http.ResponseWriter, r *http.Request) {
	if err := t.ExecuteTemplate(w, "admin.html", nil); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}
func signup(w http.ResponseWriter, r *http.Request) {
	if err := t.ExecuteTemplate(w, "signup.html", nil); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}
func leaderboard(w http.ResponseWriter, r *http.Request) {
	if err := t.ExecuteTemplate(w, "leaderboard.html", nil); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
	}
}

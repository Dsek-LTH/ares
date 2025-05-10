package main

import (
	"fmt"
	"killer-game/components"
	"net/http"

	"github.com/a-h/templ"
)

// go:embed components/*
func main() {
	router := http.NewServeMux()
	router.Handle("/{$}", templ.Handler(components.Index("test")))
	router.Handle("/admin", templ.Handler(components.Admin()))
	router.Handle("/sign-up", templ.Handler(components.Signup()))
	router.Handle("/leaderboard", templ.Handler(components.Leaderboard()))

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = server.ListenAndServe()
}

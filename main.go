package main

import (
	"database/sql"
	"embed"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"net/http"
)

type User struct {
	StilId   string `gorm:"primaryKey"`
	ImageUrl string `gorm:"not null"`
	Name     string `gorm:"not null"`
}

type Admin struct {
	UserId string `gorm:"primaryKey"`
	User   User   `gorm:"foreignKey:UserId;references:StilId"`
}

type Hunt struct {
	HunterId string `gorm:"primaryKey"`
	TargetId string `gorm:"primaryKey"`
	VideoUrl *string
	KilledAt sql.NullTime
	Hunter   User `gorm:"foreignKey:HunterId;references:StilId"`
	Target   User `gorm:"foreignKey:TargetId;references:StilId"`
}

//go:embed views/*
var views embed.FS
var t = template.Must(template.ParseFS(views, "views/*"))

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	_ = db.AutoMigrate(&User{})
	_ = db.AutoMigrate(&Admin{})
	_ = db.AutoMigrate(&Hunt{})

	// Routes
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", index)
	router.HandleFunc("GET /admin", admin)
	router.HandleFunc("GET /sign-up", signup)
	router.HandleFunc("GET /leaderboard", leaderboard)

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

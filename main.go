package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	// "github.com/a-h/templ"
	"github.com/Dsek-LTH/ares/components"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

type signUpData struct {
	Name   string `json:"name"`
	StilId string `json:"stil-id"`
}

type Server struct {
	Database *gorm.DB
}

func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	components.Index("aaa").Render(r.Context(), w)
}

func (s *Server) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var name string
	var stilId string
	var createdNewAccount bool
	if r.Method == http.MethodPost {
		var data signUpData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		// FIXME: This can error, plz fix (try Create().Error to see if error)
		s.Database.Create(User{Name: data.Name, ImageUrl: "/" + data.StilId, StilId: data.StilId})
		name = data.Name
		stilId = data.StilId
		createdNewAccount = true

	} else {
		var user User
		s.Database.Last(&user)
		name = user.Name
		stilId = user.StilId
	}
	// FIXME: This can also error, fix error handling here
	components.Signup(name, stilId, createdNewAccount).Render(r.Context(), w)
}

func (s *Server) AdminHandler(w http.ResponseWriter, r *http.Request) {
	components.Admin().Render(r.Context(), w)
}

func (s *Server) LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	components.Leaderboard().Render(r.Context(), w)
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	server := &Server{
		Database: db,
	}
	// Migrate the schema
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Admin{})
	db.AutoMigrate(&Hunt{})

	// Routes
	router := http.NewServeMux()
	router.HandleFunc("/{$}", server.IndexHandler)
	router.HandleFunc("/admin", server.AdminHandler)
	router.HandleFunc("/sign-up", server.SignUpHandler)
	router.HandleFunc("/leaderboard", server.LeaderboardHandler)

	WebServer := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = WebServer.ListenAndServe()
}

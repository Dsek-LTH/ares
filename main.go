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

type indexHandler struct {
	db       *gorm.DB
	username string
}

type signUpHandler struct {
	db                *gorm.DB
	createdNewAccount bool
	name              string
	stilId            string
}

type leaderboardHandler struct {
	db *gorm.DB
}

type adminHandler struct {
	db *gorm.DB
}

func (ih indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ih.username = "aaa"
	components.Index(ih.username).Render(r.Context(), w)
}

func (sh signUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var data signUpData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		// FIXME: This can error, plz fix (try Create().Error to see if error)
		sh.db.Create(User{Name: data.Name, ImageUrl: "/" + data.StilId, StilId: data.StilId})
		sh.name = data.Name
		sh.stilId = data.StilId
		sh.createdNewAccount = true

	} else {
		var user User
		sh.db.Last(&user)
		sh.name = user.Name
		sh.stilId = user.StilId
	}
	// FIXME: This can also error, fix error handling here
	components.Signup(sh.name, sh.stilId, sh.createdNewAccount).Render(r.Context(), w)
}

func (ah adminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	components.Admin().Render(r.Context(), w)
}

func (lh leaderboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	components.Leaderboard().Render(r.Context(), w)
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Admin{})
	db.AutoMigrate(&Hunt{})

	// Routes
	router := http.NewServeMux()
	router.Handle("/{$}", indexHandler{db: db})
	router.Handle("/admin", adminHandler{db})
	router.Handle("/sign-up", signUpHandler{db: db})
	router.Handle("/leaderboard", leaderboardHandler{db})

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = server.ListenAndServe()
}

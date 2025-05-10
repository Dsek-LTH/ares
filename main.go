package main

import (
	"database/sql"
	"fmt"
	"github.com/a-h/templ"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"killer-game/components"
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

type indexHandler struct {
	Username string
}

type signUpHandler struct {
}

type leaderboardHandler struct {
}

type adminHandler struct {
}

func newIndexHandler() indexHandler {
	return indexHandler{"<Username goes here>"}
}

func newSignUpHandler() signUpHandler {
	return signUpHandler{}
}

func newLeaderboardHandler() signUpHandler {
	return signUpHandler{}
}

func newAdminHandler() signUpHandler {
	return signUpHandler{}
}

func (ih indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	components.Index(ih.Username).Render(r.Context(), w)
}

func (sh signUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	components.Signup().Render(r.Context(), w)
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
	router.Handle("/{$}", newIndexHandler())
	router.Handle("/admin", newAdminHandler())
	router.Handle("/sign-up", newSignUpHandler())
	router.Handle("/leaderboard", newLeaderboardHandler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	fmt.Println("Server is running at localhost:8080")
	_ = server.ListenAndServe()
}

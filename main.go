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

func signUpHandler(w http.ResponseWriter, r *http.Request) {
	components.Index("aaa").Render(r.Context(), w)
}

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
	router.Handle("/{$}", signUpHandler)
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

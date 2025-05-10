package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"killer-game/components"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sessionStore *sessions.CookieStore
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
	username string
}

type signUpHandler struct {
	createdNewAccount bool
	name              string
	stilId            string
}

type leaderboardHandler struct {
}

type adminHandler struct {
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth-session")
		_, ok := session.Values["username"]

		if !ok {
			// Not logged in
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-session")
	username, ok := session.Values["username"].(string)
	log.Println(session.Values)
	if ok {
		components.Index(username).Render(r.Context(), w)
	} else {
		components.Index("Not Logged In").Render(r.Context(), w)
	}
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")
	issuer := os.Getenv("ISSUER")

	sessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Fatalf("Failed to discover provider: %v", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Admin{})
	db.AutoMigrate(&Hunt{})

	// Routes
	router := http.NewServeMux()
	router.HandleFunc("/{$}", IndexHandler)
	router.Handle("/admin", adminHandler{})
	router.Handle("GET /sign-up", adminHandler{})
	router.Handle("PUT /sign-up", signUpHandler{})
	router.Handle("/leaderboard", leaderboardHandler{})

	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, oauth2Config.AuthCodeURL("some-random-state"), http.StatusFound)
	})

	router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth-session")
		session.Options.MaxAge = -1
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)
	})

	router.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.URL.Query().Get("state") != "some-random-state" {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		oauth2Token, err := oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			return
		}

		// Extract the ID Token from OAuth2 token
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
			return
		}

		// Verify ID Token
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID token", http.StatusInternalServerError)
			return
		}

		// Extract claims
		var claims map[string]interface{}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
			return
		}

		// Save to session
		session, _ := sessionStore.Get(r, "auth-session")
		log.Println(claims["preferred_username"])
		session.Values["username"] = claims["preferred_username"]
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, fmt.Sprintf("Session store error: %s", err.Error()), http.StatusInternalServerError)
		}

		http.Redirect(w, r, "/", http.StatusFound)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Printf("Server is running at localhost:8080")
	log.Fatal(server.ListenAndServe())
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	// "strconv"

	"github.com/Dsek-LTH/ares/components"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sessionStore *sessions.CookieStore
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
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

type Handler struct {
	Database *gorm.DB
}

type DbHunterStats struct {
	UserId string
	Count  int
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth-session")
		_, ok := session.Values["username"]

		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-session")
	username, ok := session.Values["username"].(string)
	log.Println(session.Values)
	if ok {
		components.Home(username).Render(r.Context(), w)
	} else {
		components.Index().Render(r.Context(), w)
	}
}

func (s *Handler) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var data signUpData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	// FIXME: This can error, plz fix (try Create().Error to see if error)
	s.Database.Create(User{Name: data.Name, ImageUrl: "/" + data.StilId, StilId: data.StilId})
	components.Signup(data.Name, data.StilId, true).Render(r.Context(), w)
}

func (s *Handler) ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	s.Database.Last(&user)
	name := user.Name
	stilId := user.StilId
	// FIXME: This can also error, fix error handling here
	components.Signup(name, stilId, false).Render(r.Context(), w)
}

func (s *Handler) AdminHandler(w http.ResponseWriter, r *http.Request) {
	components.Admin().Render(r.Context(), w)
}

func (s *Handler) LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// alive := s.Database.

	/// get all alive people:
	// SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;

	/// get stats for all hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL GROUP BY hunter_id;

	/// get stats for all alive hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;

	var result []User
	// s.Database.Raw("SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;").Scan(&result)
	s.Database.Debug().Raw("SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;").Scan(&result)
	for _, stat := range result {
		println("id: " + stat.StilId + ", name: " + stat.Name)

	}
	components.Leaderboard().Render(r.Context(), w)
}

func (s *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauth2Config.AuthCodeURL("some-random-state"), http.StatusFound)
}

func (s *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
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
	var claims map[string]any
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

	verifier = provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})
	oauth2Config = &oauth2.Config{
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

	handler := &Handler{
		Database: db,
	}

	router := http.NewServeMux()
	router.HandleFunc("/{$}", handler.IndexHandler)
	router.Handle("/admin", authMiddleware(http.HandlerFunc(handler.AdminHandler)))
	router.HandleFunc("/sign-up", handler.SignUpHandler)
	router.HandleFunc("/leaderboard", handler.LeaderboardHandler)

	router.HandleFunc("/login", handler.LoginHandler)
	router.HandleFunc("/logout", handler.LogoutHandler)
	router.HandleFunc("/callback", handler.CallbackHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Printf("Server is running at localhost:8080")
	log.Fatal(server.ListenAndServe())
}

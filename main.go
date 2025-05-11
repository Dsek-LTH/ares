package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Dsek-LTH/ares/components"
	"github.com/Dsek-LTH/ares/db"
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
	issuer       string
	clientID     string
	clientSecret string
)

type Handler struct {
	Database *gorm.DB
}

type DbHunterStats struct {
	UserId string
	Count  int
}
type contextKey string

const userKey = contextKey("user")

// TODO: authMiddleware with options that either redirects to login page, or shows different page without redirect
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "auth-session")

		// rawIDToken, idOk := session.Values["id_token"].(string)
		accessToken, accOk := session.Values["access_token"].(string)

		if !accOk {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// _, err := verifier.Verify(r.Context(), rawIDToken)
		// if err != nil {
		// 	// Expired or invalid token
		// 	session.Options.MaxAge = -1
		// 	session.Save(r, w)
		// 	http.Redirect(w, r, "/login", http.StatusFound)
		// 	return
		// }

		// Overkill?
		active, err := introspectToken(accessToken)
		if err != nil || !active {
			// Invalid or expired token — force re-login
			session.Options.MaxAge = -1 // clear session
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		username, ok := session.Values["username"]

		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
		} else {
			ctx := context.WithValue(r.Context(), userKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func introspectToken(accessToken string) (bool, error) {
	url := strings.TrimSuffix(issuer, "/") + "/protocol/openid-connect/token/introspect"
	req, _ := http.NewRequest("POST",
		url,
		strings.NewReader("token="+accessToken))

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Active bool `json:"active"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)

	return result.Active, err
}

func (s *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "auth-session")
	username, ok := session.Values["username"].(string)

	if ok {
		components.Home(username).Render(r.Context(), w)
	} else {
		components.Index().Render(r.Context(), w)
	}
}

func (s *Handler) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var data db.SignUpData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	// FIXME: This can error, plz fix (try Create().Error to see if error)
	s.Database.Create(db.User{Name: data.Name, ImageUrl: "/" + data.StilId, StilId: data.StilId})
	components.Signup(data.Name, data.StilId, true).Render(r.Context(), w)
}

func (s *Handler) ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	var user db.User
	s.Database.Last(&user)
	name := user.Name
	stilId := user.StilId
	// FIXME: This can also error, fix error handling here
	components.Signup(name, stilId, false).Render(r.Context(), w)
}

func (s *Handler) AdminHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(userKey).(string)
	components.Admin(username).Render(r.Context(), w)
}

func (s *Handler) LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// alive := s.Database.

	/// get all alive people:
	// SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;

	/// get stats for all hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL GROUP BY hunter_id;

	/// get stats for all alive hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;

	var result []db.User
	// s.Database.Raw("SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;").Scan(&result)
	s.Database.Debug().Raw("SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;").Scan(&result)
	for _, stat := range result {
		println("id: " + stat.StilId + ", name: " + stat.Name)

	}
	components.Leaderboard().Render(r.Context(), w)
}

func (s *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// FIXME: some-random-state should probably be changed...
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
	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Exp               int64  `json:"exp"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	// Save to session
	session, _ := sessionStore.Get(r, "auth-session")
	// Without an external service like Redis, only id_token or access_token may be saved.
	// For complexity's sake, only access_token is saved to verify the session using the introspect endpoint.
	// session.Values["id_token"] = rawIDToken
	session.Values["username"] = claims.PreferredUsername
	session.Values["expires_at"] = claims.Exp
	session.Values["access_token"] = oauth2Token.AccessToken
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Session store error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	clientID = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")
	issuer = os.Getenv("ISSUER")

	sessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

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

	db_con, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database")
	}

	// Migrate the schema
	db_con.AutoMigrate(&db.User{})
	db_con.AutoMigrate(&db.Admin{})
	db_con.AutoMigrate(&db.Hunt{})

	handler := &Handler{
		Database: db_con,
	}

	router := http.NewServeMux()
	router.HandleFunc("/{$}", handler.IndexHandler)
	router.Handle("/admin", authMiddleware(http.HandlerFunc(handler.AdminHandler)))
	router.Handle("/sign-up", authMiddleware(http.HandlerFunc(handler.SignUpHandler)))
	router.Handle("/leaderboard", authMiddleware(http.HandlerFunc(handler.LeaderboardHandler)))

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

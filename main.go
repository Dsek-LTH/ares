package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Dsek-LTH/ares/db"
	"github.com/Dsek-LTH/ares/handler"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DbHunterStats struct {
	UserId string
	Count  int
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

	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
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

	db_con, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database")
	}

	// Migrate the schema
	db_con.AutoMigrate(&db.User{}, &db.Admin{}, &db.Hunt{})

	h := &handler.Handler{
		Database:     db_con,
		SessionStore: sessionStore,
		OAuth2Vals: handler.OAuth2Vals{
			Issuer:       issuer,
			Verifier:     verifier,
			Oauth2Config: oauth2Config,
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
	}

	router := http.NewServeMux()
	router.Handle("/{$}", h.AuthMiddleware(http.HandlerFunc(h.IndexHandler), false))
	router.Handle("/admin", h.AuthMiddleware(http.HandlerFunc(h.AdminHandler), true))
	router.Handle("/sign-up", h.AuthMiddleware(http.HandlerFunc(h.ShowUserHandler), true))
	router.Handle("/leaderboard", h.AuthMiddleware(http.HandlerFunc(h.LeaderboardHandler), true))

	router.HandleFunc("/login", h.LoginHandler)
	router.HandleFunc("/logout", h.LogoutHandler)
	router.HandleFunc("/callback", h.CallbackHandler)

	router.Handle("/assets/",
		http.StripPrefix("/assets",
			http.FileServer(http.Dir("assets"))))

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Printf("Server is running at localhost:8080")
	log.Fatal(server.ListenAndServe())
}

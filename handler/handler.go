package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Dsek-LTH/ares/components"
	"github.com/Dsek-LTH/ares/db"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type contextKey string

const UserKey = contextKey("user")

type OAuth2Vals struct {
	Issuer       string
	Verifier     *oidc.IDTokenVerifier
	Oauth2Config *oauth2.Config
	ClientID     string
	ClientSecret string
}

type Handler struct {
	Database     *gorm.DB
	SessionStore *sessions.CookieStore
	OAuth2Vals   OAuth2Vals
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(UserKey)

	if username == nil {
		components.Index().Render(r.Context(), w)
	} else {
		user := username.(string)
		components.Home(user).Render(r.Context(), w)
	}
}

func (h *Handler) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	var data db.SignUpData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	// FIXME: This can error, plz fix (try Create().Error to see if error)
	h.Database.Create(db.User{Name: data.Name, ImageUrl: "/" + data.StilId, StilId: data.StilId})
	components.Signup(data.Name, data.StilId, true).Render(r.Context(), w)
}

func (h *Handler) ShowUserHandler(w http.ResponseWriter, r *http.Request) {
	var user db.User
	h.Database.Last(&user)
	name := user.Name
	stilId := user.StilId
	// FIXME: This can also error, fix error handling here
	components.Signup(name, stilId, false).Render(r.Context(), w)
}

func (h *Handler) AdminHandler(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(UserKey).(string)
	components.Admin(username).Render(r.Context(), w)
}

func (h *Handler) LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// alive := s.Database.

	/// get all alive people:
	// SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;

	/// get stats for all hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL GROUP BY hunter_id;

	/// get stats for all alive hunters:
	// SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;

	var result []db.User
	// s.Database.Raw("SELECT hunter_id, COUNT(killed_at) FROM hunts WHERE killed_at IS NOT NULL AND hunter_id IN (SELECT DISTINCT target_id FROM hunts WHERE killed_at IS NULL) GROUP BY hunter_id;").Scan(&result)
	h.Database.Debug().Raw("SELECT * from users join hunts on users.stil_id = hunts.target_id WHERE killed_at IS NULL;").Scan(&result)
	for _, stat := range result {
		println("id: " + stat.StilId + ", name: " + stat.Name)

	}
	components.Leaderboard().Render(r.Context(), w)
}

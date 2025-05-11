package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RequireAuth makes automatic redirects to /login if true, and a context value username with a possible value of nil if false
type RequireAuth bool

// TODO: authMiddleware with options that either redirects to login page, or shows different page without redirect
func (h *Handler) AuthMiddleware(next http.HandlerFunc, requireAuth RequireAuth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := h.SessionStore.Get(r, "auth-session")
		accessToken, ok := session.Values["access_token"].(string)

		if !ok || accessToken == "" {
			if requireAuth {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Overkill?
		active, err := introspectToken(accessToken, &h.OAuth2Vals)
		if err != nil || !active {
			// Invalid or expired token — force re-login
			session.Options.MaxAge = -1 // clear session
			session.Save(r, w)

			if requireAuth {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		username, ok := session.Values["username"]

		if !ok {
			if requireAuth {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		ctx := context.WithValue(r.Context(), UserKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func introspectToken(accessToken string, oVals *OAuth2Vals) (bool, error) {
	url := strings.TrimSuffix(oVals.Issuer, "/") + "/protocol/openid-connect/token/introspect"
	req, _ := http.NewRequest("POST",
		url,
		strings.NewReader("token="+accessToken))

	req.SetBasicAuth(oVals.ClientID, oVals.ClientSecret)
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

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// FIXME: some-random-state should probably be changed...
	http.Redirect(w, r, h.OAuth2Vals.Oauth2Config.AuthCodeURL("some-random-state"), http.StatusFound)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.SessionStore.Get(r, "auth-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.URL.Query().Get("state") != "some-random-state" {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	oauth2Token, err := h.OAuth2Vals.Oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
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
	idToken, err := h.OAuth2Vals.Verifier.Verify(ctx, rawIDToken)
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
	session, _ := h.SessionStore.Get(r, "auth-session")
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

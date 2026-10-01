package tesla

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	state, nonce := randomString(16), randomString(16)
	a.setCookie(w, "oauth_state", state, 600)
	url := a.config.OAuth.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("prompt_missing_scopes", "true"))
	http.Redirect(w, r, url, http.StatusFound)
}

func (a *App) callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value == "" || cookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	a.setCookie(w, "oauth_state", "", -1)
	if oauthError := r.URL.Query().Get("error"); oauthError != "" {
		http.Error(w, "authorization error: "+oauthError+" "+r.URL.Query().Get("error_description"), http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := a.config.OAuth.Exchange(r.Context(), code, oauth2.SetAuthURLParam("audience", a.config.Audience))
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			log.Printf("token exchange failed: %s %s", retrieveErr.ErrorCode, retrieveErr.Body)
		}
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}
	if token.RefreshToken == "" {
		log.Println("warning: Tesla did not return a refresh token")
	}
	userID := randomString(16)
	a.store.Save(userID, token)
	a.setCookie(w, "user_id", userID, 60*60*24*90)
	http.Redirect(w, r, "/me", http.StatusFound)
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	source, err := a.tokenSource(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	client := oauth2.NewClient(r.Context(), source)
	response, err := client.Get(strings.TrimRight(a.config.Audience, "/") + "/api/1/users/me")
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) && (retrieveErr.ErrorCode == "login_required" || retrieveErr.Response.StatusCode == http.StatusUnauthorized) {
			a.forgetUser(cookie.Value)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func (a *App) setCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode})
}

func randomString(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

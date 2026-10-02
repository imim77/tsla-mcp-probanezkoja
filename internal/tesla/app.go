package tesla

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// App owns the OAuth HTTP flow and token lifecycle.
type App struct {
	config  Config
	store   TokenStore
	secure  bool
	mu      sync.Mutex
	sources map[string]*persistingSource
}

// NewApp creates an application using an in-memory token store.
func NewApp(config Config) *App {
	return &App{config: config, store: newMemoryTokenStore(), secure: strings.HasPrefix(config.OAuth.RedirectURL, "https://"), sources: make(map[string]*persistingSource)}
}

// Handler returns the application's HTTP routes.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	if len(a.config.PublicKeyPEM) > 0 {
		mux.HandleFunc("/.well-known/appspecific/com.tesla.3p.public-key.pem", a.publicKey)
	}
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/auth/callback", a.callback)
	mux.HandleFunc("/me", a.me)
	return mux
}

func (a *App) publicKey(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.config.PublicKeyPEM)
}

func (a *App) tokenSource(userID string) (oauth2.TokenSource, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if source, ok := a.sources[userID]; ok {
		return source, nil
	}
	token, ok := a.store.Get(userID)
	if !ok {
		return nil, errors.New("no token for user")
	}

	base := a.config.OAuth.TokenSource(context.Background(), token)
	source := &persistingSource{userID: userID, store: a.store, source: oauth2.ReuseTokenSource(token, base), last: token}
	a.sources[userID] = source
	return source, nil
}

func (a *App) forgetUser(userID string) {
	a.store.Delete(userID)
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sources, userID)
}

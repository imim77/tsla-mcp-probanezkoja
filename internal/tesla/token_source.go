package tesla

import (
	"sync"

	"golang.org/x/oauth2"
)

// persistingSource records rotated refresh tokens after each refresh.
type persistingSource struct {
	mu     sync.Mutex
	userID string
	store  TokenStore
	source oauth2.TokenSource
	last   *oauth2.Token
}

func (s *persistingSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, err := s.source.Token()
	if err != nil {
		return nil, err
	}
	if s.last == nil || token.AccessToken != s.last.AccessToken || token.RefreshToken != s.last.RefreshToken {
		s.store.Save(s.userID, token)
		s.last = token
	}
	return token, nil
}

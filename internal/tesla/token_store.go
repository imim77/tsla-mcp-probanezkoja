package tesla

import (
	"sync"

	"golang.org/x/oauth2"
)

// TokenStore persists OAuth tokens for application users.
type TokenStore interface {
	Get(userID string) (*oauth2.Token, bool)
	Save(userID string, token *oauth2.Token)
	Delete(userID string)
}

type memoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*oauth2.Token
}

func newMemoryTokenStore() *memoryTokenStore {
	return &memoryTokenStore{tokens: make(map[string]*oauth2.Token)}
}

func (s *memoryTokenStore) Get(userID string) (*oauth2.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[userID]
	return token, ok
}

func (s *memoryTokenStore) Save(userID string, token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[userID] = token
}

func (s *memoryTokenStore) Delete(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, userID)
}

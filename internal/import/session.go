package importutil

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ImportSession stores temporary data for a multi-step import process.
type ImportSession struct {
	ID        string
	Filename  string
	Data      *ParsedData
	Mapping   *FieldMapping
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore manages import sessions with automatic cleanup.
type SessionStore struct {
	sessions map[string]*ImportSession
	mu       sync.RWMutex
	ttl      time.Duration
}

// NewSessionStore creates a new session store with the specified TTL.
func NewSessionStore(ttl time.Duration) *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*ImportSession),
		ttl:      ttl,
	}
	// Start background cleanup
	go store.cleanup()
	return store
}

// Create creates a new import session and returns its ID.
func (s *SessionStore) Create(filename string, data *ParsedData) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := generateSessionID()
	now := time.Now()

	session := &ImportSession{
		ID:        sessionID,
		Filename:  filename,
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
	}

	s.sessions[sessionID] = session
	return sessionID
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(sessionID string) (*ImportSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// Update updates an existing session.
func (s *SessionStore) Update(sessionID string, mapping *FieldMapping) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return false
	}

	session.Mapping = mapping
	return true
}

// Delete removes a session from the store.
func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
}

// cleanup periodically removes expired sessions.
func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// generateSessionID creates a random session ID.
func generateSessionID() string {
	const sessionIDBytes = 16
	b := make([]byte, sessionIDBytes)
	_, _ = rand.Read(b) // crypto/rand.Read always returns n == len(b) and err == nil
	return hex.EncodeToString(b)
}

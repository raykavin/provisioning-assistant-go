package services

import (
	"provisioning-assistant/internal/domain"
	"sync"
	"time"
)

type SessionService struct {
	sessions map[int64]*domain.Session
	mu       sync.RWMutex
}

// NewSessionService creates a new session service instance
func NewSessionService() *SessionService {
	return &SessionService{
		sessions: make(map[int64]*domain.Session),
	}
}

// CreateSession creates a new user session with idle state
func (s *SessionService) CreateSession(userID, chatID int64) *domain.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &domain.Session{
		UserID:    userID,
		ChatID:    chatID,
		State:     domain.StateIdle,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.sessions[userID] = session
	return session
}

// GetSession retrieves a session by user ID, returns nil if expired
func (s *SessionService) GetSession(userID int64) *domain.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if session, exists := s.sessions[userID]; exists {
		if time.Since(session.UpdatedAt) > 30*time.Minute {
			delete(s.sessions, userID)
			return nil
		}
		return session
	}
	return nil
}

// UpdateSession updates session timestamp and saves changes
func (s *SessionService) UpdateSession(session *domain.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.UpdatedAt = time.Now()
	s.sessions[session.UserID] = session
}

// DeleteSession removes a session from memory
func (s *SessionService) DeleteSession(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, userID)
}

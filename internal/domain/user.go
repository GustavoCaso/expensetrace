package domain

import "time"

type Session interface {
	ID() string
	UserID() int64
	ExpiresAt() time.Time
	CreatedAt() time.Time
}

type User interface {
	ID() int64
	Username() string
	PasswordHash() string
	CreatedAt() time.Time
}

type user struct {
	id           int64
	username     string
	passwordHash string
	createdAt    time.Time
}

func (u *user) ID() int64 {
	return u.id
}

func (u *user) Username() string {
	return u.username
}

func (u *user) PasswordHash() string {
	return u.passwordHash
}

func (u *user) CreatedAt() time.Time {
	return u.createdAt
}

func NewUser(id int64, username, passwordHash string, createdAt time.Time) User {
	return &user{
		id:           id,
		username:     username,
		passwordHash: passwordHash,
		createdAt:    createdAt,
	}
}

type session struct {
	id        string
	userID    int64
	expiresAt time.Time
	createdAt time.Time
}

func (s *session) ID() string {
	return s.id
}

func (s *session) UserID() int64 {
	return s.userID
}

func (s *session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s *session) CreatedAt() time.Time {
	return s.createdAt
}

func NewSession(id string, userID int64, expiresAt, createdAt time.Time) Session {
	return &session{
		id:        id,
		userID:    userID,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}
}

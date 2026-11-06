package domain

import "time"

// APIToken is a long-lived bearer token a user creates to let external
// clients (e.g. AI agents over MCP) read their data. Only a hash of the
// token is ever stored; the plaintext is shown to the user once at creation.
type APIToken interface {
	ID() int64
	UserID() int64
	Name() string
	CreatedAt() time.Time
	LastUsedAt() *time.Time
}

type apiToken struct {
	id         int64
	userID     int64
	name       string
	createdAt  time.Time
	lastUsedAt *time.Time
}

func (t *apiToken) ID() int64 {
	return t.id
}

func (t *apiToken) UserID() int64 {
	return t.userID
}

func (t *apiToken) Name() string {
	return t.name
}

func (t *apiToken) CreatedAt() time.Time {
	return t.createdAt
}

func (t *apiToken) LastUsedAt() *time.Time {
	return t.lastUsedAt
}

func NewAPIToken(id, userID int64, name string, createdAt time.Time, lastUsedAt *time.Time) APIToken {
	return &apiToken{
		id:         id,
		userID:     userID,
		name:       name,
		createdAt:  createdAt,
		lastUsedAt: lastUsedAt,
	}
}

// ProfileViewData is the view data for the profile page.
type ProfileViewData struct {
	ViewBase
	APITokens []APIToken
	// MCPURL is the endpoint agents should connect to.
	MCPURL string
}

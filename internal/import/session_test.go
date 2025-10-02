package importutil

import (
	"testing"
	"time"
)

func TestSessionStoreCreate(t *testing.T) {
	store := NewSessionStore(10 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source", "date", "description", "amount", "currency"},
		Rows:    [][]string{{"Bank A", "01/01/2024", "Coffee", "-5.00", "USD"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)
	if sessionID == "" {
		t.Fatal("Create returned empty session ID")
	}

	// Verify session was created
	session, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Session not found after creation")
	}

	if session.ID != sessionID {
		t.Errorf("Session.ID = %q, want %q", session.ID, sessionID)
	}
	if session.Filename != "test.csv" {
		t.Errorf("Session.Filename = %q, want 'test.csv'", session.Filename)
	}
	if session.Data != data {
		t.Error("Session.Data does not match created data")
	}
	if session.Mapping != nil {
		t.Error("Session.Mapping should be nil initially")
	}
}

func TestSessionStoreGet(t *testing.T) {
	store := NewSessionStore(10 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	// Get existing session
	session, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Get failed to retrieve existing session")
	}
	if session.ID != sessionID {
		t.Errorf("Retrieved session ID = %q, want %q", session.ID, sessionID)
	}

	// Get non-existent session
	_, exists = store.Get("non-existent-id")
	if exists {
		t.Error("Get returned true for non-existent session")
	}
}

func TestSessionStoreUpdate(t *testing.T) {
	store := NewSessionStore(10 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	// Update session with mapping
	mapping := &FieldMapping{
		Source:            "Test Bank",
		DateColumn:        1,
		DescriptionColumn: 2,
		AmountColumn:      3,
		CurrencyColumn:    4,
	}

	success := store.Update(sessionID, mapping)
	if !success {
		t.Fatal("Update failed for existing session")
	}

	// Verify mapping was stored
	session, _ := store.Get(sessionID)
	if session.Mapping != mapping {
		t.Error("Session.Mapping was not updated")
	}

	// Update non-existent session
	success = store.Update("non-existent-id", mapping)
	if success {
		t.Error("Update returned true for non-existent session")
	}
}

func TestSessionStoreDelete(t *testing.T) {
	store := NewSessionStore(10 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	// Verify session exists
	_, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Session should exist before deletion")
	}

	// Delete session
	store.Delete(sessionID)

	// Verify session was deleted
	_, exists = store.Get(sessionID)
	if exists {
		t.Error("Session still exists after deletion")
	}

	// Delete non-existent session (should not panic)
	store.Delete("non-existent-id")
}

func TestSessionStoreExpiration(t *testing.T) {
	// Use very short TTL for testing
	store := NewSessionStore(100 * time.Millisecond)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	// Session should exist immediately
	_, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Session should exist immediately after creation")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired
	_, exists = store.Get(sessionID)
	if exists {
		t.Error("Session should be expired after TTL")
	}
}

func TestSessionStoreCleanup(t *testing.T) {
	// Use short TTL for testing
	store := NewSessionStore(50 * time.Millisecond)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	// Create multiple sessions
	id1 := store.Create("test1.csv", data)
	id2 := store.Create("test2.csv", data)

	// Both should exist
	_, exists := store.Get(id1)
	if !exists {
		t.Fatal("Session 1 should exist")
	}
	_, exists = store.Get(id2)
	if !exists {
		t.Fatal("Session 2 should exist")
	}

	// Wait for cleanup (runs every minute, but TTL is 50ms)
	time.Sleep(100 * time.Millisecond)

	// Sessions should be expired when we try to get them
	_, exists = store.Get(id1)
	if exists {
		t.Error("Session 1 should be expired")
	}
	_, exists = store.Get(id2)
	if exists {
		t.Error("Session 2 should be expired")
	}
}

func TestSessionStoreConcurrency(t *testing.T) {
	store := NewSessionStore(1 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	// Create sessions concurrently
	const numGoroutines = 10
	done := make(chan bool)
	sessionIDs := make(chan string, numGoroutines)

	for range numGoroutines {
		go func() {
			id := store.Create("test.csv", data)
			sessionIDs <- id
			done <- true
		}()
	}

	// Wait for all goroutines
	for range numGoroutines {
		<-done
	}
	close(sessionIDs)

	// Verify all sessions were created
	count := 0
	for id := range sessionIDs {
		_, exists := store.Get(id)
		if !exists {
			t.Errorf("Session %s not found", id)
		}
		count++
	}

	if count != 10 {
		t.Errorf("Expected 10 sessions, got %d", count)
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple session IDs and ensure they're unique
	const numTestIDs = 100
	const expectedIDLength = 32 // 16 bytes = 32 hex characters
	ids := make(map[string]bool)

	for range numTestIDs {
		id := generateSessionID()
		if id == "" {
			t.Error("generateSessionID returned empty string")
		}
		if len(id) != expectedIDLength {
			t.Errorf("generateSessionID returned ID of length %d, want %d", len(id), expectedIDLength)
		}
		if ids[id] {
			t.Errorf("generateSessionID returned duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestSessionStoreMultipleUpdates(t *testing.T) {
	store := NewSessionStore(10 * time.Minute)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	// Update multiple times
	mapping1 := &FieldMapping{
		Source:            "Test Bank",
		DateColumn:        1,
		DescriptionColumn: 2,
		AmountColumn:      3,
		CurrencyColumn:    4,
	}

	mapping2 := &FieldMapping{
		Source:            "Another Bank",
		DateColumn:        0,
		DescriptionColumn: 3,
		AmountColumn:      2,
		CurrencyColumn:    4,
	}

	store.Update(sessionID, mapping1)
	session, _ := store.Get(sessionID)
	if session.Mapping != mapping1 {
		t.Error("First mapping not stored correctly")
	}

	store.Update(sessionID, mapping2)
	session, _ = store.Get(sessionID)
	if session.Mapping != mapping2 {
		t.Error("Second mapping not stored correctly (should overwrite first)")
	}
}

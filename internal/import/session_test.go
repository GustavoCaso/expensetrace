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

	session, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Get failed to retrieve existing session")
	}
	if session.ID != sessionID {
		t.Errorf("Retrieved session ID = %q, want %q", session.ID, sessionID)
	}

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

	session, _ := store.Get(sessionID)
	if session.Mapping != mapping {
		t.Error("Session.Mapping was not updated")
	}

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

	_, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Session should exist before deletion")
	}

	store.Delete(sessionID)

	_, exists = store.Get(sessionID)
	if exists {
		t.Error("Session still exists after deletion")
	}

	store.Delete("non-existent-id")
}

func TestSessionStoreExpiration(t *testing.T) {
	store := NewSessionStore(100 * time.Millisecond)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	sessionID := store.Create("test.csv", data)

	_, exists := store.Get(sessionID)
	if !exists {
		t.Fatal("Session should exist immediately after creation")
	}

	time.Sleep(150 * time.Millisecond)

	_, exists = store.Get(sessionID)
	if exists {
		t.Error("Session should be expired after TTL")
	}
}

func TestSessionStoreCleanup(t *testing.T) {
	store := NewSessionStore(50 * time.Millisecond)

	data := &ParsedData{
		Headers: []string{"source"},
		Rows:    [][]string{{"Bank A"}},
		Format:  "csv",
	}

	id1 := store.Create("test1.csv", data)
	id2 := store.Create("test2.csv", data)

	_, exists := store.Get(id1)
	if !exists {
		t.Fatal("Session 1 should exist")
	}
	_, exists = store.Get(id2)
	if !exists {
		t.Fatal("Session 2 should exist")
	}

	time.Sleep(100 * time.Millisecond)

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

	for range numGoroutines {
		<-done
	}
	close(sessionIDs)

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
	const numTestIDs = 100
	const expectedIDLength = 32
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

package util

import "testing"

func TestGenerateRandomID(t *testing.T) {
	const iterations = 10000
	const idLength = 16
	seen := make(map[string]bool)

	for i := range iterations {
		randomID := GenerateRandomID(idLength)

		// Check for collisions - should NEVER happen with crypto/rand
		if seen[randomID] {
			t.Fatalf("Session ID collision detected after %d iterations. "+
				"This indicates use of predictable PRNG instead of crypto/rand. "+
				"Session ID: %s", i+1, randomID)
		}

		seen[randomID] = true

		// Verify length is appropriate
		if len(randomID) < 32 {
			t.Errorf("Session ID too short: %d characters (expected at least 32)", len(randomID))
		}
	}
}

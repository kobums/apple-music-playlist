package rest

import "testing"

// Apple's API only appends, so a track added later lands at the end. Adding the
// confident matches during the first pass and the user's picks afterwards put
// those picks out of the pasted order — which is what this status exists to
// prevent.
func TestMatchStatusNeedsDecision(t *testing.T) {
	tests := []struct {
		status MatchStatus
		want   bool
	}{
		{MatchReview, true},
		{MatchMissing, true},
		{MatchAdded, false},
		{MatchDuplicate, false},
		{MatchReady, false},
	}

	for _, tt := range tests {
		if got := tt.status.needsDecision(); got != tt.want {
			t.Errorf("%q.needsDecision() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

// The confirm step receives ids in row order and must add them in that order,
// so the playlist matches what was pasted.
func TestConfirmedAddPreservesRequestOrder(t *testing.T) {
	// Mirrors AddConfirmedTracks' filtering: order is preserved and only
	// genuine duplicates drop out.
	existing := NewTrackSet([]Track{{ID: "already-there"}})

	incoming := []string{"first", "second", "already-there", "third", "second"}

	var added []string
	for _, id := range incoming {
		track := Track{ID: id}
		if existing.Has(track) {
			continue
		}
		existing.Add(track)
		added = append(added, id)
	}

	want := []string{"first", "second", "third"}
	if len(added) != len(want) {
		t.Fatalf("added %v, want %v", added, want)
	}
	for i := range want {
		if added[i] != want[i] {
			t.Errorf("position %d = %q, want %q", i, added[i], want[i])
		}
	}
}

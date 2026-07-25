package rest

import "testing"

// AddConfirmedTracks only has catalog ids to work with, so it builds
// Track{ID: ...} with no artist or title. Those must be compared by id alone.
//
// Add() used to write the empty name key unconditionally, so confirming the
// first pick poisoned the name set and every later pick in the same request
// came back as "already there".
func TestTrackSetIDOnlyTracksDoNotCollide(t *testing.T) {
	set := NewTrackSet(nil)

	first := Track{ID: "1443621832"}
	second := Track{ID: "1383158341"}
	third := Track{ID: "603198228"}

	if set.Has(first) {
		t.Fatal("empty set reported the first track as present")
	}
	set.Add(first)

	if reason, found := set.Find(second); found {
		t.Errorf("second id-only track reported as duplicate by %s", reason)
	}
	set.Add(second)

	if reason, found := set.Find(third); found {
		t.Errorf("third id-only track reported as duplicate by %s", reason)
	}

	// The same id twice must still be caught.
	if _, found := set.Find(first); !found {
		t.Error("a repeated id should be reported as duplicate")
	}
}

// A library entry with neither artist nor title must not match everything.
func TestTrackSetBlankLibraryEntryMatchesNothing(t *testing.T) {
	set := NewTrackSet([]Track{{ID: "", Name: "", ArtistName: ""}})

	for _, track := range []Track{
		{ID: "111"},
		{ID: "222", Name: "밤편지", ArtistName: "아이유"},
		{Name: "붉은 노을", ArtistName: "이문세"},
	} {
		if reason, found := set.Find(track); found {
			t.Errorf("%+v matched a blank library entry by %s", track, reason)
		}
	}
}

// Mixing full tracks and id-only tracks in one request must stay correct.
func TestTrackSetMixedFullAndIDOnly(t *testing.T) {
	set := NewTrackSet([]Track{{ID: "999", Name: "밤편지", ArtistName: "아이유"}})

	// Already present by id.
	if _, found := set.Find(Track{ID: "999"}); !found {
		t.Error("id-only lookup missed a track already in the playlist")
	}

	// Not present, id only.
	if _, found := set.Find(Track{ID: "111"}); found {
		t.Error("id-only lookup falsely matched")
	}

	set.Add(Track{ID: "111"})

	// A different full track must still be judged on its own merits.
	if reason, found := set.Find(Track{ID: "222", Name: "Citizen Kane", ArtistName: "혁오"}); found {
		t.Errorf("unrelated full track reported as duplicate by %s", reason)
	}
}

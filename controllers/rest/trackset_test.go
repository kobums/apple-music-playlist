package rest

import "testing"

func libraryTrack(id, name, artist string) Track {
	return Track{ID: id, Name: name, ArtistName: artist}
}

// The playlist's existing contents were never read, so re-running the same list
// appended every song again — and adding to an existing playlist is the app's
// main use.
func TestTrackSetMatchesByCatalogID(t *testing.T) {
	set := NewTrackSet([]Track{libraryTrack("1440857781", "밤편지", "아이유")})

	if !set.Has(libraryTrack("1440857781", "Night Letter", "IU")) {
		t.Error("same catalog id should count as present regardless of naming")
	}
	if set.Has(libraryTrack("9999999999", "Blueming", "아이유")) {
		t.Error("a different song must not be treated as present")
	}
}

// Library entries that were never linked to the catalog carry no catalog id,
// so name matching is the only thing that can catch them.
func TestTrackSetMatchesByNameWhenIDMissing(t *testing.T) {
	set := NewTrackSet([]Track{libraryTrack("", "붉은 노을", "이문세")})

	if !set.Has(libraryTrack("1234567890", "붉은 노을", "이문세")) {
		t.Error("same artist and title should count as present")
	}
	if set.Has(libraryTrack("1234567890", "붉은 노을", "빅뱅")) {
		t.Error("same title by a different artist is a different recording")
	}
}

// Normalisation means a re-release under a slightly different string still
// counts as the same recording.
func TestTrackSetIgnoresFormattingDifferences(t *testing.T) {
	set := NewTrackSet([]Track{libraryTrack("", "Love wins all (feat. someone)", "IU")})

	if !set.Has(libraryTrack("", "Love Wins All", "IU")) {
		t.Error("featuring clause and casing should not create a false new track")
	}
}

// A song listed twice in one paste must go in once.
func TestTrackSetAddCatchesDuplicatesWithinOneRequest(t *testing.T) {
	set := NewTrackSet(nil)
	track := libraryTrack("1440857781", "밤편지", "아이유")

	if set.Has(track) {
		t.Fatal("an empty set should contain nothing")
	}

	set.Add(track)
	if !set.Has(track) {
		t.Error("Add should make the track count as present")
	}
}

func TestTrackSetEmptyAndNilAreSafe(t *testing.T) {
	set := NewTrackSet(nil)
	if set.Has(libraryTrack("1", "a", "b")) {
		t.Error("empty set matched something")
	}

	// A track with neither id nor name must not match every blank entry.
	set.Add(Track{})
	if set.Has(libraryTrack("2", "밤편지", "아이유")) {
		t.Error("a blank entry must not swallow real tracks")
	}

	var nilSet *TrackSet
	if nilSet.Has(libraryTrack("1", "a", "b")) {
		t.Error("nil set should report nothing present")
	}
	nilSet.Add(libraryTrack("1", "a", "b")) // must not panic
}

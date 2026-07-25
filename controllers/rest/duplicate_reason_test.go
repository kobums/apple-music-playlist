package rest

import "testing"

// Which rule fired matters when diagnosing a false positive: a catalog-id match
// is exact, while a name match is a heuristic that can over-reach.
func TestTrackSetFindReportsWhichRuleMatched(t *testing.T) {
	set := NewTrackSet([]Track{
		{ID: "1440857781", Name: "밤편지", ArtistName: "아이유"},
		{ID: "", Name: "붉은 노을", ArtistName: "이문세"},
	})

	tests := []struct {
		name       string
		track      Track
		wantReason DuplicateReason
		wantFound  bool
	}{
		{
			name:       "exact catalog id",
			track:      Track{ID: "1440857781", Name: "Night Letter", ArtistName: "IU"},
			wantReason: DuplicateByCatalogID,
			wantFound:  true,
		},
		{
			name:       "name match when the library entry has no catalog id",
			track:      Track{ID: "9999999999", Name: "붉은 노을", ArtistName: "이문세"},
			wantReason: DuplicateByName,
			wantFound:  true,
		},
		{
			name:      "unrelated track",
			track:     Track{ID: "5555555555", Name: "Bohemian Rhapsody", ArtistName: "Queen"},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, found := set.Find(tt.track)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if found && reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

// A fresh playlist must report nothing as present. This is the case the user
// hit: an empty playlist that answered "already there".
func TestEmptyPlaylistHasNoDuplicates(t *testing.T) {
	set := NewTrackSet(nil)

	for _, track := range []Track{
		{ID: "1", Name: "Maria Elena", ArtistName: "사비에르 쿠가트"},
		{ID: "2", Name: "앞으로 (feat. 안다영)", ArtistName: "아프로"},
		{ID: "3", Name: "세상이 멈출 때까지", ArtistName: "호지"},
		{ID: "4", Name: "夢中人", ArtistName: "왕페이"},
		{ID: "5", Name: "레이디", ArtistName: "CHS"},
		{ID: "6", Name: "4랑을 4랑해", ArtistName: "shinjihang, 윤다혜 & 그냥노창"},
	} {
		if reason, found := set.Find(track); found {
			t.Errorf("%s - %s reported as duplicate by %s in an empty playlist",
				track.ArtistName, track.Name, reason)
		}
	}
}

// The real tracks from the report must not collide with each other, which is
// what a within-request false positive would look like.
func TestReportedTracksDoNotCollideWithEachOther(t *testing.T) {
	tracks := []Track{
		{ID: "1", Name: "The Look of Love", ArtistName: "버트 바카락"},
		{ID: "2", Name: "Kowloon Hong Kong", ArtistName: "潘迪華"},
		{ID: "3", Name: "Citizen Kane", ArtistName: "혁오"},
		{ID: "4", Name: "Maria Elena", ArtistName: "사비에르 쿠가트"},
		{ID: "5", Name: "Perfidia", ArtistName: "Xavier Cugat y su Orquesta"},
		{ID: "6", Name: "Happy Together", ArtistName: "The Turtles"},
		{ID: "7", Name: "앞으로 (feat. 안다영)", ArtistName: "아프로"},
		{ID: "8", Name: "4랑을 4랑해", ArtistName: "shinjihang, 윤다혜 & 그냥노창"},
		{ID: "9", Name: "세상이 멈출 때까지", ArtistName: "호지"},
		{ID: "10", Name: "夢中人", ArtistName: "왕페이"},
		{ID: "11", Name: "Heart Of Glass", ArtistName: "Blondie"},
		{ID: "12", Name: "White Sky", ArtistName: "Vampire Weekend"},
		{ID: "13", Name: "레이디", ArtistName: "CHS"},
		{ID: "14", Name: "Hollywood", ArtistName: "검정치마"},
		{ID: "15", Name: "Places We Won't Walk", ArtistName: "Bruno Major"},
	}

	set := NewTrackSet(nil)
	for _, track := range tracks {
		if reason, found := set.Find(track); found {
			t.Errorf("%s - %s collided with an earlier track (by %s)",
				track.ArtistName, track.Name, reason)
		}
		set.Add(track)
	}

	// The same recording listed twice must still be caught.
	if _, found := set.Find(Track{ID: "9", Name: "세상이 멈출 때까지", ArtistName: "호지"}); !found {
		t.Error("a repeated track should be reported as duplicate")
	}
}

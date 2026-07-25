package rest

import "testing"

// normalize() strips bracketed annotations, so the acoustic/instrumental take
// and the studio original collapsed to identical text and the second one was
// dropped as a duplicate. They are different recordings and both belong in the
// playlist.
func TestDifferentRecordingsAreNotDuplicates(t *testing.T) {
	original := Track{ID: "1", Name: "세상이 멈출 때까지", ArtistName: "호지"}
	acoustic := Track{ID: "2", Name: "세상이 멈출 때까지 (Acoustic Ver.) [Instrumental]", ArtistName: "호지"}

	set := NewTrackSet([]Track{original})

	if reason, found := set.Find(acoustic); found {
		t.Errorf("acoustic/instrumental take reported as duplicate of the original by %s", reason)
	}

	// And the other way round.
	reversed := NewTrackSet([]Track{acoustic})
	if reason, found := reversed.Find(original); found {
		t.Errorf("original reported as duplicate of the acoustic take by %s", reason)
	}
}

// The same variant twice must still collapse — that is the whole point.
func TestSameVariantStillDeduplicates(t *testing.T) {
	tests := []struct{ name, a, b string }{
		{"identical titles", "붉은 노을", "붉은 노을"},
		{"same live take, different spacing", "붉은 노을 (Live)", "붉은 노을  (live)"},
		{"same instrumental", "밤편지 [Instrumental]", "밤편지 (Instrumental)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewTrackSet([]Track{{Name: tt.a, ArtistName: "아무개"}})
			if _, found := set.Find(Track{Name: tt.b, ArtistName: "아무개"}); !found {
				t.Errorf("%q and %q should be the same recording", tt.a, tt.b)
			}
		})
	}
}

// Each variant is distinct from the others, not just from the original.
func TestVariantsAreDistinctFromEachOther(t *testing.T) {
	variants := []string{
		"세상이 멈출 때까지",
		"세상이 멈출 때까지 (Live)",
		"세상이 멈출 때까지 (Acoustic Ver.)",
		"세상이 멈출 때까지 (Instrumental)",
		"세상이 멈출 때까지 (Remix)",
	}

	set := NewTrackSet(nil)
	for _, title := range variants {
		track := Track{Name: title, ArtistName: "호지"}
		if reason, found := set.Find(track); found {
			t.Errorf("%q collided with an earlier variant by %s", title, reason)
		}
		set.Add(track)
	}
}

// "mr" used to be a variant marker and matched the honorific, so a plain pop
// song was treated as a karaoke backing track — penalised while scoring and
// keyed as a variant while deduplicating.
func TestHonorificIsNotAVariantMarker(t *testing.T) {
	for _, title := range []string{"Mr. Brightside", "Mr. Blue Sky", "Mr. Rain"} {
		if markers := markersIn(title); len(markers) > 0 {
			t.Errorf("%q flagged as variant %v", title, markers)
		}
	}
}

// ASCII markers match on word boundaries, so a marker must not surface inside
// an unrelated word.
func TestMarkersMatchWholeWordsOnly(t *testing.T) {
	tests := []struct {
		text      string
		wantEmpty bool
	}{
		{"Alive", true},            // not "live"
		{"Instrumental", false},    // real marker
		{"Delivery", true},         // not "live"
		{"Recovery", true},         // not "cover"
		{"Live at Wembley", false}, // real marker
	}

	for _, tt := range tests {
		markers := markersIn(tt.text)
		if tt.wantEmpty && len(markers) > 0 {
			t.Errorf("%q flagged as variant %v", tt.text, markers)
		}
		if !tt.wantEmpty && len(markers) == 0 {
			t.Errorf("%q should have been flagged as a variant", tt.text)
		}
	}
}

// "inst" must not fire inside "instrumental", so the key stays predictable.
func TestInstMarkerDoesNotFireInsideInstrumental(t *testing.T) {
	markers := markersIn("밤편지 (Instrumental)")
	if markers["inst"] {
		t.Error(`"inst" matched inside "instrumental"`)
	}
	if !markers["instrumental"] {
		t.Error(`"instrumental" was not detected`)
	}
}

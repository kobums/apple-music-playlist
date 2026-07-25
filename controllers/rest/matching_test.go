package rest

import "testing"

func track(name, artist string) Track {
	return Track{ID: name + "|" + artist, Name: name, ArtistName: artist}
}

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"BIGBANG", "bigbang"},
		{"붉은 노을 (Remix)", "붉은 노을"},
		{"Love wins all (feat. someone)", "love wins all"},
		{"Blueming - Single Version", "blueming single version"},
		{"  Spaced   Out  ", "spaced out"},
		{"Don't Stop!", "don t stop"},
	}
	for _, tt := range tests {
		if got := normalize(tt.in); got != tt.want {
			t.Errorf("normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBestMatchExactPairAutoAdds(t *testing.T) {
	candidates := []Track{
		track("붉은 노을", "빅뱅"),
		track("붉은 노을", "이문세"),
	}

	got := BestMatch("빅뱅", "붉은 노을", candidates)
	if got.Status != MatchAdded {
		t.Fatalf("status = %q (score %.2f), want %q", got.Status, got.Score, MatchAdded)
	}
	if got.Track.ArtistName != "빅뱅" {
		t.Errorf("picked artist %q, want 빅뱅", got.Track.ArtistName)
	}
}

// A wrong-but-first hit is exactly what the old limit=1 behaviour shipped.
func TestBestMatchPrefersRightArtistOverFirstHit(t *testing.T) {
	candidates := []Track{
		track("밤편지", "누군가"),
		track("밤편지", "아이유"),
	}

	got := BestMatch("아이유", "밤편지", candidates)
	if got.Track == nil || got.Track.ArtistName != "아이유" {
		t.Fatalf("picked %+v, want the 아이유 track", got.Track)
	}
}

// "제목 - 아티스트" is common in pasted lists and used to be searched as-is.
func TestBestMatchHandlesReversedOrder(t *testing.T) {
	candidates := []Track{track("밤편지", "아이유")}

	got := BestMatch("밤편지", "아이유", candidates)
	if got.Status != MatchAdded {
		t.Errorf("reversed input: status = %q (score %.2f), want %q", got.Status, got.Score, MatchAdded)
	}
}

// Asking for the studio take and getting a live one must not auto-add.
func TestBestMatchDemotesUnrequestedVariants(t *testing.T) {
	live := BestMatch("빅뱅", "붉은 노을", []Track{track("붉은 노을 (Live)", "빅뱅")})
	if live.Status == MatchAdded {
		t.Errorf("live take auto-added with score %.2f; want review", live.Score)
	}

	inst := BestMatch("아이유", "밤편지", []Track{track("밤편지 (Instrumental)", "아이유")})
	if inst.Status == MatchAdded {
		t.Errorf("instrumental auto-added with score %.2f; want review", inst.Score)
	}
}

// ...but a live version that was actually asked for should still match.
func TestBestMatchKeepsRequestedVariant(t *testing.T) {
	got := BestMatch("빅뱅", "붉은 노을 (Live)", []Track{track("붉은 노을 (Live)", "빅뱅")})
	if got.Status != MatchAdded {
		t.Errorf("requested live take: status = %q (score %.2f), want %q", got.Status, got.Score, MatchAdded)
	}
}

func TestBestMatchUnrelatedIsMissing(t *testing.T) {
	got := BestMatch("아이유", "밤편지", []Track{track("Bohemian Rhapsody", "Queen")})
	if got.Status != MatchMissing {
		t.Errorf("status = %q (score %.2f), want %q", got.Status, got.Score, MatchMissing)
	}
	if len(got.Candidates) == 0 {
		t.Error("candidates should still be returned so the user can look")
	}
}

func TestBestMatchNoCandidates(t *testing.T) {
	got := BestMatch("아이유", "밤편지", nil)
	if got.Status != MatchMissing {
		t.Errorf("status = %q, want %q", got.Status, MatchMissing)
	}
	if got.Track != nil {
		t.Error("no candidates should yield no track")
	}
}

// Real candidate list returned by the KR catalog for "빅뱅 붉은 노을". Five
// different artists share the title, which is exactly the case the old limit=1
// search could not survive: whichever Apple happened to rank first was added
// and reported as a success.
func realRedSunsetCandidates() []Track {
	return []Track{
		track("붉은 노을", "BIGBANG"),
		track("붉은 노을", "이문세"),
		track("붉은 노을 (원곡가수 이문세)", "YB"),
		track("붉은 노을 (Live)", "이문세"),
		track("붉은 노을", "윤하"),
	}
}

// Artists whose catalog name is written the same way the user wrote it must be
// picked on merit and added without asking.
func TestBestMatchDisambiguatesSharedTitle(t *testing.T) {
	for _, artist := range []string{"이문세", "윤하"} {
		t.Run(artist, func(t *testing.T) {
			got := BestMatch(artist, "붉은 노을", realRedSunsetCandidates())
			if got.Track == nil {
				t.Fatalf("no track chosen (score %.2f)", got.Score)
			}
			if got.Track.ArtistName != artist {
				t.Errorf("chose %q, want %q", got.Track.ArtistName, artist)
			}
			// 이문세 has both a studio and a live take here; the studio one must
			// win when no live version was asked for.
			if got.Track.Name != "붉은 노을" {
				t.Errorf("chose recording %q, want the plain studio version", got.Track.Name)
			}
			if got.Status != MatchAdded {
				t.Errorf("status = %q (score %.2f), want %q", got.Status, got.Score, MatchAdded)
			}
		})
	}
}

// "빅뱅" and "BIGBANG" are the same act, but nothing local can know that — the
// artist similarity is 0 and every candidate sharing the title scores alike.
//
// Two things must hold. Apple's ranking breaks the tie so the right track is
// still offered, and the score stays low enough that it is offered rather than
// added, because we genuinely cannot verify the artist. Silently adding here is
// the exact failure this feature exists to prevent.
func TestBestMatchRomanisedArtistGoesToReview(t *testing.T) {
	got := BestMatch("빅뱅", "붉은 노을", realRedSunsetCandidates())

	if got.Track == nil || got.Track.ArtistName != "BIGBANG" {
		t.Fatalf("chose %+v, want Apple's top-ranked BIGBANG track", got.Track)
	}
	if got.Status != MatchReview {
		t.Errorf("status = %q (score %.2f), want %q", got.Status, got.Score, MatchReview)
	}
	if len(got.Candidates) != len(realRedSunsetCandidates()) {
		t.Errorf("got %d candidates, want all of them for the user to choose from", len(got.Candidates))
	}
}

// The tie-break must not be strong enough to promote a weak match, otherwise it
// would reintroduce "whatever Apple returned first wins".
func TestRankBonusCannotReachAutoAdd(t *testing.T) {
	if rankBonus(0) >= autoAddScore-reviewScore {
		t.Fatalf("rankBonus(0) = %.3f is large enough to change a verdict", rankBonus(0))
	}
	for i := range 6 {
		if rankBonus(i) < 0 {
			t.Errorf("rankBonus(%d) = %.3f, must never be negative", i, rankBonus(i))
		}
	}
}

// Spacing and romanisation drift should not block a match.
func TestBestMatchToleratesSpacingDrift(t *testing.T) {
	got := BestMatch("big bang", "haru haru", []Track{track("Haru Haru", "BIGBANG")})
	if got.Status == MatchMissing {
		t.Errorf("status = %q (score %.2f), want a match", got.Status, got.Score)
	}
}

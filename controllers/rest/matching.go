package rest

import (
	"regexp"
	"strings"
)

// Confidence thresholds for turning a score into a decision.
//
// The old behaviour was to take the first search hit unconditionally, so a
// cover, a live take or a same-named artist was added silently and the user saw
// "added" either way. A wrong track is worse than a missing one because nothing
// signals it, so anything below autoAddScore is now handed back for the user to
// confirm instead of being guessed at.
const (
	autoAddScore = 0.86
	reviewScore  = 0.45
)

// MatchStatus is the outcome for one input line.
type MatchStatus string

const (
	MatchAdded     MatchStatus = "added"
	MatchDuplicate MatchStatus = "duplicate"
	MatchReview    MatchStatus = "review"
	MatchMissing   MatchStatus = "missing"
)

// TrackSet answers "is this recording already in the playlist?".
//
// Two keys are kept because neither alone is enough. The catalog id is exact
// but absent from library entries that were never matched to the catalog; the
// normalized name catches those, and also the case where the same song was
// added earlier from a different release.
type TrackSet struct {
	catalogIDs map[string]bool
	names      map[string]bool
}

// trackKey normalizes "artist + title" into a comparable identity, or "" when
// there is nothing to compare.
//
// Returning "" for a blank pair matters: callers that only have a catalog id
// (confirming a user's pick) pass an otherwise empty Track, and a shared blank
// key would make every one of them look like the same recording.
func trackKey(artist, title string) string {
	normalizedArtist, normalizedTitle := normalize(artist), normalize(title)
	if normalizedArtist == "" && normalizedTitle == "" {
		return ""
	}
	return normalizedArtist + "\x00" + normalizedTitle
}

func NewTrackSet(tracks []Track) *TrackSet {
	set := &TrackSet{
		catalogIDs: make(map[string]bool, len(tracks)),
		names:      make(map[string]bool, len(tracks)),
	}
	for _, track := range tracks {
		if track.ID != "" {
			set.catalogIDs[track.ID] = true
		}
		if key := trackKey(track.ArtistName, track.Name); key != "" {
			set.names[key] = true
		}
	}
	return set
}

// DuplicateReason says which key matched, so a false positive can be traced to
// the exact rule that caused it.
type DuplicateReason string

const (
	DuplicateByCatalogID DuplicateReason = "catalog-id"
	DuplicateByName      DuplicateReason = "name"
)

// Find reports whether the playlist already contains this recording, and why.
func (s *TrackSet) Find(track Track) (DuplicateReason, bool) {
	if s == nil {
		return "", false
	}
	if track.ID != "" && s.catalogIDs[track.ID] {
		return DuplicateByCatalogID, true
	}
	if key := trackKey(track.ArtistName, track.Name); key != "" && s.names[key] {
		return DuplicateByName, true
	}
	return "", false
}

// Has reports whether the playlist already contains this recording.
func (s *TrackSet) Has(track Track) bool {
	_, found := s.Find(track)
	return found
}

// Add records a track so duplicates inside one request are caught too — the
// same song listed twice in the pasted text must not be added twice.
func (s *TrackSet) Add(track Track) {
	if s == nil {
		return
	}
	if track.ID != "" {
		s.catalogIDs[track.ID] = true
	}
	// The guard was missing here while NewTrackSet had it, so confirming a pick
	// that carried only a catalog id wrote a blank key that then matched every
	// later id-only track.
	if key := trackKey(track.ArtistName, track.Name); key != "" {
		s.names[key] = true
	}
}

var (
	bracketed   = regexp.MustCompile(`[(\[{][^)\]}]*[)\]}]`)
	featuring   = regexp.MustCompile(`(?i)\s*(feat\.?|ft\.?|featuring|with)\s+.*$`)
	punctuation = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	spaces      = regexp.MustCompile(`\s+`)
)

// variantMarkers are words that change *which recording* a track is. A studio
// original and its live take share a title, so without this a live version
// scores as a perfect match.
var variantMarkers = []string{
	"live", "라이브", "실황",
	"remix", "리믹스",
	"instrumental", "inst", "mr", "반주",
	"cover", "커버",
	"karaoke", "노래방",
	"acoustic", "어쿠스틱",
	"remaster", "remastered", "리마스터",
	"sped up", "slowed", "nightcore",
	"radio edit", "extended",
}

// normalize reduces a title or artist to comparable text: no bracketed
// annotations, no featuring clause, no punctuation, single-spaced, lowercase.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = bracketed.ReplaceAllString(s, " ")
	s = featuring.ReplaceAllString(s, " ")
	s = punctuation.ReplaceAllString(s, " ")
	s = spaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// markersIn reports which variant markers appear anywhere in the raw text.
func markersIn(s string) map[string]bool {
	lower := strings.ToLower(s)
	found := make(map[string]bool)
	for _, marker := range variantMarkers {
		if strings.Contains(lower, marker) {
			found[marker] = true
		}
	}
	return found
}

func levenshtein(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// similarity returns 0..1 for two already-normalized strings. Edit distance is
// used rather than exact equality so romanisation and spacing differences
// ("bigbang" vs "big bang") still match.
func similarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	ra, rb := []rune(a), []rune(b)
	longest := max(len(ra), len(rb))
	score := 1 - float64(levenshtein(ra, rb))/float64(longest)

	// One side fully containing the other is a strong signal that edit distance
	// under-rates, e.g. "bigbang" inside "bigbang the band".
	if strings.Contains(a, b) || strings.Contains(b, a) {
		score = max(score, 0.9)
	}
	return max(score, 0)
}

// scorePair rates one candidate against a parsed artist/title pair.
//
// Title carries more weight than artist: users mistype or abbreviate artist
// names far more often than song titles, and the catalog's artist field may
// list only the lead of a collaboration.
func scorePair(queryArtist, queryTitle string, candidate Track) float64 {
	titleScore := similarity(normalize(queryTitle), normalize(candidate.Name))
	artistScore := similarity(normalize(queryArtist), normalize(candidate.ArtistName))

	score := 0.6*titleScore + 0.4*artistScore

	// Penalise a candidate that is a different recording than what was asked
	// for. Asking for a live version and getting one is fine; asking for the
	// original and getting a live take is not.
	queryMarkers := markersIn(queryArtist + " " + queryTitle)
	for marker := range markersIn(candidate.Name + " " + candidate.AlbumName) {
		if !queryMarkers[marker] {
			score -= 0.25
			break
		}
	}

	return max(score, 0)
}

// rankBonus is a small preference for candidates Apple ranked higher, used only
// to break ties between locally indistinguishable candidates. It is deliberately
// far smaller than the gap between review and auto-add so it can reorder
// equals, never promote a weak match into a confident one.
func rankBonus(index int) float64 {
	return max(0, 0.06-float64(index)*0.015)
}

// Match is the chosen candidate for one input line, with everything the UI
// needs to let a person confirm or override it.
type Match struct {
	Status     MatchStatus `json:"status"`
	Score      float64     `json:"score"`
	Track      *Track      `json:"track,omitempty"`
	Candidates []Track     `json:"candidates,omitempty"`
}

// BestMatch picks the strongest candidate for "artist - title".
//
// Both orderings are scored because "제목 - 아티스트" is common in pasted lists
// and used to search as-is, producing confident-looking nonsense. Whichever
// reading the catalog agrees with more wins.
func BestMatch(artist, title string, candidates []Track) Match {
	if len(candidates) == 0 {
		return Match{Status: MatchMissing}
	}

	bestIndex, bestScore, bestRanked := 0, -1.0, -1.0
	for i, candidate := range candidates {
		score := max(
			scorePair(artist, title, candidate),
			scorePair(title, artist, candidate),
		)

		// Selection uses Apple's ranking as a tiebreaker; confidence does not.
		//
		// Local scoring cannot tell that "빅뱅" and "BIGBANG" are the same act,
		// so every artist sharing a title scores identically and the winner
		// would come down to slice order. Apple's relevance ranking does know,
		// and it is the only signal available. Keeping the bonus out of
		// bestScore means an unverifiable artist still lands in review rather
		// than being silently added on Apple's say-so.
		ranked := score + rankBonus(i)
		if ranked > bestRanked {
			bestIndex, bestScore, bestRanked = i, score, ranked
		}
	}

	best := candidates[bestIndex]

	switch {
	case bestScore >= autoAddScore:
		return Match{Status: MatchAdded, Score: bestScore, Track: &best, Candidates: candidates}
	case bestScore >= reviewScore:
		return Match{Status: MatchReview, Score: bestScore, Track: &best, Candidates: candidates}
	default:
		return Match{Status: MatchMissing, Score: bestScore, Candidates: candidates}
	}
}

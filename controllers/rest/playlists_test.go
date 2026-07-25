package rest

import "testing"

func libPlaylist(id, name string, canEdit bool, dateAdded string) libraryPlaylist {
	var playlist libraryPlaylist
	playlist.ID = id
	playlist.Attributes.Name = name
	playlist.Attributes.CanEdit = canEdit
	playlist.Attributes.DateAdded = dateAdded
	return playlist
}

// Apple-owned playlists cannot be written to. Offering them would only fail at
// save time, after the user has already picked one and pasted a song list.
func TestEditablePlaylistsHidesReadOnlyOnes(t *testing.T) {
	got := editablePlaylists([]libraryPlaylist{
		libPlaylist("p.mine", "여름 밤", true, "2026-07-01T00:00:00Z"),
		libPlaylist("p.apple", "New Music Daily", false, "2026-07-02T00:00:00Z"),
		libPlaylist("p.other", "드라이브", true, "2026-07-03T00:00:00Z"),
	})

	for _, playlist := range got {
		if playlist.ID == "p.apple" {
			t.Error("a read-only playlist was offered as a target")
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d playlists, want 2 editable ones", len(got))
	}
}

// Newest first: the playlist someone just made is usually the one they mean.
func TestEditablePlaylistsSortsNewestFirst(t *testing.T) {
	got := editablePlaylists([]libraryPlaylist{
		libPlaylist("old", "오래된", true, "2024-01-01T00:00:00Z"),
		libPlaylist("newest", "최근", true, "2026-07-20T00:00:00Z"),
		libPlaylist("middle", "중간", true, "2025-05-05T00:00:00Z"),
	})

	want := []string{"newest", "middle", "old"}
	if len(got) != len(want) {
		t.Fatalf("got %d playlists, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Errorf("position %d = %q, want %q", i, got[i].ID, want[i])
		}
	}
}

// Entries without an id are unusable, and a missing dateAdded must not panic or
// drop the playlist.
func TestEditablePlaylistsSkipsUnusableEntries(t *testing.T) {
	got := editablePlaylists([]libraryPlaylist{
		libPlaylist("", "id 없음", true, "2026-07-01T00:00:00Z"),
		libPlaylist("p.ok", "날짜 없음", true, ""),
	})

	if len(got) != 1 || got[0].ID != "p.ok" {
		t.Fatalf("got %+v, want only p.ok", got)
	}
	if got[0].Name != "날짜 없음" {
		t.Errorf("name = %q, want 날짜 없음", got[0].Name)
	}
}

func TestEditablePlaylistsEmptyLibrary(t *testing.T) {
	if got := editablePlaylists(nil); len(got) != 0 {
		t.Errorf("got %d playlists from an empty library", len(got))
	}
}

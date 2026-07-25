package rest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeKey(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("not a real key"), 0o400); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

func TestResolvePrivateKeyPath(t *testing.T) {
	t.Run("PRIVATE_KEY_PATH wins over anything on disk", func(t *testing.T) {
		dir := t.TempDir()
		writeKey(t, dir, "AuthKey_ONDISK0001.p8")
		t.Setenv("PRIVATE_KEY_PATH", "/explicit/AuthKey_CHOSEN.p8")

		got, err := resolvePrivateKeyPath(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/explicit/AuthKey_CHOSEN.p8" {
			t.Errorf("got %q, want the explicit path", got)
		}
	})

	t.Run("a single key on disk is used without configuration", func(t *testing.T) {
		dir := t.TempDir()
		want := writeKey(t, dir, "AuthKey_5BWCYC2764.p8")
		t.Setenv("PRIVATE_KEY_PATH", "")

		got, err := resolvePrivateKeyPath(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	// After a team switch both the old and new key tend to sit side by side.
	// Silently picking one would sign tokens with whichever sorts first, so this
	// has to fail loudly instead.
	t.Run("several keys is an error naming them", func(t *testing.T) {
		dir := t.TempDir()
		writeKey(t, dir, "AuthKey_AAAAAAAAAA.p8")
		writeKey(t, dir, "AuthKey_BBBBBBBBBB.p8")
		t.Setenv("PRIVATE_KEY_PATH", "")

		_, err := resolvePrivateKeyPath(dir)
		if err == nil {
			t.Fatal("expected an error when several keys are present")
		}
		for _, want := range []string{"AuthKey_AAAAAAAAAA.p8", "AuthKey_BBBBBBBBBB.p8", "PRIVATE_KEY_PATH"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not mention %q", err, want)
			}
		}
	})

	t.Run("no key is an error saying what to do", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("PRIVATE_KEY_PATH", "")

		_, err := resolvePrivateKeyPath(dir)
		if err == nil {
			t.Fatal("expected an error when no key is present")
		}
		if !strings.Contains(err.Error(), "PRIVATE_KEY_PATH") {
			t.Errorf("error %q should tell the operator how to fix it", err)
		}
	})
}

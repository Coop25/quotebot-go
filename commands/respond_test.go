package commands

import (
	"encoding/json"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is t…"},
		{"ééééé", 3, "éé…"}, // counts characters, not bytes
	}
	for _, tt := range tests {
		if got := truncate(tt.in, tt.max); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
	}
}

func TestNoMentionsSerializesToEmptyParse(t *testing.T) {
	b, err := json.Marshal(noMentions)
	if err != nil {
		t.Fatal(err)
	}
	// An explicit empty "parse" list is what tells Discord not to ping
	// anyone; null or a missing field falls back to Discord's defaults.
	want := `{"parse":[],"roles":[],"users":[],"replied_user":false}`
	if string(b) != want {
		t.Fatalf("got %s, want %s", b, want)
	}
}

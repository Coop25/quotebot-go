package commands

import (
	"encoding/json"
	"testing"
)

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

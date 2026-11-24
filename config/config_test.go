package config

import (
	"reflect"
	"testing"
)

func TestParseAllowedChannels(t *testing.T) {
	got, err := parseAllowedChannels(" quote:123 , multi-quote: 456 ,")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"quote": "123", "multi-quote": "456"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got, err = parseAllowedChannels("")
	if err != nil || len(got) != 0 {
		t.Errorf("empty value: got %v, %v", got, err)
	}

	for _, bad := range []string{"quote", "quote:", ":123", "quote:123,roll"} {
		if _, err := parseAllowedChannels(bad); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}

package commands

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/google/uuid"
)

var fixedExportTime = time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

func TestExportRoundTrip(t *testing.T) {
	quotes := []postgres.Quote{
		{ID: uuid.New(), GuildID: "111", Quote: "first"},
		{ID: uuid.New(), GuildID: "111", Quote: "second\nwith a newline"},
		{ID: uuid.New(), GuildID: "222", Quote: "émoji 🎉"},
	}

	data, err := encodeExport(quotes, fixedExportTime)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version": 1`, `"exported_at": "2026-11-24T10:00:00Z"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("export JSON missing %s:\n%s", want, data)
		}
	}

	got, err := decodeExport(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, quotes) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, quotes)
	}
}

func TestExportFileName(t *testing.T) {
	if got, want := exportFileName(fixedExportTime), "quotebot-export-2026-11-24.json"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeExportEmpty(t *testing.T) {
	got, err := decodeExport(strings.NewReader(`{"version":1,"quotes":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no quotes, got %v", got)
	}
}

func TestDecodeExportRejects(t *testing.T) {
	id := uuid.NewString()
	tests := map[string]string{
		"not json":        "not json",
		"future version":  `{"version":2,"quotes":[]}`,
		"bad id":          `{"version":1,"quotes":[{"id":"nope","guild_id":"1","quote":"q"}]}`,
		"nil id":          `{"version":1,"quotes":[{"id":"` + uuid.Nil.String() + `","guild_id":"1","quote":"q"}]}`,
		"missing id":      `{"version":1,"quotes":[{"guild_id":"1","quote":"q"}]}`,
		"empty guild id":  `{"version":1,"quotes":[{"id":"` + id + `","guild_id":"","quote":"q"}]}`,
		"empty quote":     `{"version":1,"quotes":[{"id":"` + id + `","guild_id":"1","quote":""}]}`,
		"other json":      `{"hello":"world"}`,
		"truncated":       `{"version":1,"quotes":[{"id":"` + id,
		"over size limit": strings.Repeat(" ", maxImportBytes+1),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if got, err := decodeExport(strings.NewReader(input)); err == nil {
				t.Errorf("expected an error, got %v", got)
			}
		})
	}
}

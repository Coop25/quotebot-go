package commands

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/google/uuid"
)

func makeQuotes(texts ...string) []postgres.Quote {
	quotes := make([]postgres.Quote, len(texts))
	for i, text := range texts {
		quotes[i] = postgres.Quote{ID: uuid.New(), Quote: text}
	}
	return quotes
}

func TestFormatMultiQuote(t *testing.T) {
	quotes := makeQuotes("first", "second")

	got := formatMultiQuote(quotes, false, 2000)
	if want := "first\nsecond\n"; got != want {
		t.Errorf("without ids: got %q, want %q", got, want)
	}

	got = formatMultiQuote(quotes, true, 2000)
	want := "> **Quote ID: **" + quotes[0].ID.String() + "\n" +
		"> **Quote ID: **" + quotes[1].ID.String() + "\n" +
		"\nfirst\nsecond\n"
	if got != want {
		t.Errorf("with ids: got %q, want %q", got, want)
	}
}

func TestFormatMultiQuoteStopsAtLimit(t *testing.T) {
	quotes := makeQuotes(strings.Repeat("a", 900), strings.Repeat("b", 900), strings.Repeat("c", 900))

	for _, showID := range []bool{false, true} {
		got := formatMultiQuote(quotes, showID, 2000)
		if n := utf8.RuneCountInString(got); n > 2000 {
			t.Errorf("showID=%v: length %d exceeds limit", showID, n)
		}
		if !strings.Contains(got, "bbb") || strings.Contains(got, "ccc") {
			t.Errorf("showID=%v: expected exactly the first two quotes, got %q", showID, got)
		}
		if showID && strings.Contains(got, quotes[2].ID.String()) {
			t.Errorf("ID of the dropped quote should not be listed")
		}
	}
}

func TestFormatMultiQuoteTruncatesOversizedFirstQuote(t *testing.T) {
	quotes := makeQuotes(strings.Repeat("x", 3000))
	got := formatMultiQuote(quotes, false, 2000)
	if n := utf8.RuneCountInString(got); n != 2000 {
		t.Errorf("expected truncation to 2000 characters, got %d", n)
	}
}

package postgres

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestAllQuotesIncludesEveryGuild(t *testing.T) {
	a := newTestAccessor(t)
	guildA, guildB := newGuildID(), newGuildID()

	qa, err := a.AddQuote("quote in A", guildA)
	if err != nil {
		t.Fatal(err)
	}
	qb, err := a.AddQuote("quote in B", guildB)
	if err != nil {
		t.Fatal(err)
	}

	all, err := a.AllQuotes()
	if err != nil {
		t.Fatal(err)
	}
	found := map[uuid.UUID]Quote{}
	for _, q := range all {
		found[q.ID] = q
	}
	for _, want := range []Quote{qa, qb} {
		if got, ok := found[want.ID]; !ok || got != want {
			t.Errorf("AllQuotes: want %+v, got %+v (present=%v)", want, got, ok)
		}
	}
}

func TestImportQuotesSkipsExisting(t *testing.T) {
	a := newTestAccessor(t)
	guildID := newGuildID()
	quotes := []Quote{
		{ID: uuid.New(), GuildID: guildID, Quote: "one"},
		{ID: uuid.New(), GuildID: guildID, Quote: "two"},
		{ID: uuid.New(), GuildID: guildID, Quote: "three"},
	}

	if n, err := a.ImportQuotes(quotes); err != nil || n != 3 {
		t.Fatalf("first import: got %d, %v; want 3, nil", n, err)
	}
	if n, err := a.ImportQuotes(quotes); err != nil || n != 0 {
		t.Fatalf("second import: got %d, %v; want 0, nil", n, err)
	}
	got, err := a.GetRandomQuotes(guildID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Errorf("got %d quotes in the guild, want 3", len(got))
	}
}

func TestImportQuotesIsAllOrNothing(t *testing.T) {
	a := newTestAccessor(t)
	guildID := newGuildID()
	quotes := []Quote{
		{ID: uuid.New(), GuildID: guildID, Quote: "good"},
		// Postgres rejects NUL bytes in text, failing the second insert.
		{ID: uuid.New(), GuildID: guildID, Quote: "bad\x00"},
	}

	if n, err := a.ImportQuotes(quotes); err == nil || n != 0 {
		t.Fatalf("got %d, %v; want 0 and an error", n, err)
	}
	if _, err := a.GetRandomQuotes(guildID, 10); !errors.Is(err, ErrNoQuotes) {
		t.Errorf("the good quote should have been rolled back; GetRandomQuotes err = %v", err)
	}
}

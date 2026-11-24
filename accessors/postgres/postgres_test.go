package postgres

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
)

// newTestAccessor connects to the database in TEST_DATABASE_URL and runs the
// migrations. Tests are skipped when it is not set, so `go test ./...` works
// without a local Postgres. CI sets it.
func newTestAccessor(t *testing.T) *postgresAccessor {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}

	// Run twice: migrations run on every startup, so they must be repeatable.
	for i := 0; i < 2; i++ {
		if err := runMigrations(db, "../../migrations"); err != nil {
			t.Fatalf("running migrations (pass %d): %v", i+1, err)
		}
	}
	return &postgresAccessor{db: db}
}

// newGuildID returns a guild ID no other test uses, so tests don't see each
// other's quotes.
func newGuildID() string {
	return "test-" + uuid.NewString()
}

func TestEmptyGuildReturnsErrNoQuotes(t *testing.T) {
	a := newTestAccessor(t)
	guildID := newGuildID()

	if _, err := a.GetRandomQuote(guildID); !errors.Is(err, ErrNoQuotes) {
		t.Errorf("GetRandomQuote: got %v, want ErrNoQuotes", err)
	}
	if _, err := a.GetRandomQuotes(guildID, 5); !errors.Is(err, ErrNoQuotes) {
		t.Errorf("GetRandomQuotes: got %v, want ErrNoQuotes", err)
	}
}

func TestAddAndGetQuotes(t *testing.T) {
	a := newTestAccessor(t)
	guildID, otherGuildID := newGuildID(), newGuildID()

	added := map[uuid.UUID]string{}
	for _, text := range []string{"first quote", "second quote", "third quote"} {
		q, err := a.AddQuote(text, guildID)
		if err != nil {
			t.Fatalf("AddQuote: %v", err)
		}
		if q.Quote != text || q.GuildID != guildID || q.ID == uuid.Nil {
			t.Fatalf("AddQuote returned %+v", q)
		}
		added[q.ID] = text
	}
	if _, err := a.AddQuote("other guild's quote", otherGuildID); err != nil {
		t.Fatalf("AddQuote: %v", err)
	}

	t.Run("GetRandomQuote returns a quote from the guild", func(t *testing.T) {
		q, err := a.GetRandomQuote(guildID)
		if err != nil {
			t.Fatal(err)
		}
		if added[q.ID] != q.Quote || q.GuildID != guildID {
			t.Errorf("unexpected quote %+v", q)
		}
	})

	t.Run("GetRandomQuotes returns distinct quotes from only this guild", func(t *testing.T) {
		quotes, err := a.GetRandomQuotes(guildID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(quotes) != len(added) {
			t.Fatalf("got %d quotes, want all %d", len(quotes), len(added))
		}
		seen := map[uuid.UUID]bool{}
		for _, q := range quotes {
			if seen[q.ID] {
				t.Errorf("duplicate quote %s", q.ID)
			}
			seen[q.ID] = true
			if added[q.ID] != q.Quote || q.GuildID != guildID {
				t.Errorf("unexpected quote %+v", q)
			}
		}
	})

	t.Run("GetRandomQuotes respects the limit", func(t *testing.T) {
		quotes, err := a.GetRandomQuotes(guildID, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(quotes) != 2 {
			t.Errorf("got %d quotes, want 2", len(quotes))
		}
	})
}

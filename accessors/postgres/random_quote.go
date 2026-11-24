package postgres

import (
	"database/sql"
	"errors"
)

// ErrNoQuotes is returned when a guild has no quotes saved yet.
var ErrNoQuotes = errors.New("no quotes found")

func (a *postgresAccessor) GetRandomQuote(guildID string) (Quote, error) {
	row := a.db.QueryRow("SELECT id, quote FROM quotes WHERE guild_id = $1 ORDER BY RANDOM() LIMIT 1", guildID)
	var quote Quote
	err := row.Scan(&quote.ID, &quote.Quote)
	if errors.Is(err, sql.ErrNoRows) {
		return Quote{}, ErrNoQuotes
	}
	if err != nil {
		return Quote{}, err
	}

	quote.GuildID = guildID
	return quote, nil
}

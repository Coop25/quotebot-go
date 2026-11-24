package postgres

import (
	"database/sql"
	"errors"
	"fmt"
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

// GetRandomQuotes returns up to limit distinct random quotes for a guild.
func (a *postgresAccessor) GetRandomQuotes(guildID string, limit int) ([]Quote, error) {
	rows, err := a.db.Query("SELECT id, quote FROM quotes WHERE guild_id = $1 ORDER BY RANDOM() LIMIT $2", guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying random quotes: %w", err)
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		quote := Quote{GuildID: guildID}
		if err := rows.Scan(&quote.ID, &quote.Quote); err != nil {
			return nil, fmt.Errorf("scanning random quote: %w", err)
		}
		quotes = append(quotes, quote)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading random quotes: %w", err)
	}
	if len(quotes) == 0 {
		return nil, ErrNoQuotes
	}
	return quotes, nil
}

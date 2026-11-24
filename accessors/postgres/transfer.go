package postgres

import "fmt"

// AllQuotes returns every quote in the database, across all guilds.
func (a *postgresAccessor) AllQuotes() ([]Quote, error) {
	rows, err := a.db.Query("SELECT id, guild_id, quote FROM quotes ORDER BY guild_id, id")
	if err != nil {
		return nil, fmt.Errorf("querying all quotes: %w", err)
	}
	defer rows.Close()

	quotes := []Quote{}
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.GuildID, &q.Quote); err != nil {
			return nil, fmt.Errorf("scanning quote: %w", err)
		}
		quotes = append(quotes, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading quotes: %w", err)
	}
	return quotes, nil
}

// ImportQuotes inserts quotes in a single transaction, skipping any whose ID
// already exists. It returns how many were inserted. On error nothing is
// written.
func (a *postgresAccessor) ImportQuotes(quotes []Quote) (imported int, err error) {
	tx, err := a.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("starting import: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			imported = 0
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO quotes (id, guild_id, quote) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING")
	if err != nil {
		return 0, fmt.Errorf("preparing import: %w", err)
	}
	defer stmt.Close()

	for i, q := range quotes {
		res, err := stmt.Exec(q.ID, q.GuildID, q.Quote)
		if err != nil {
			return 0, fmt.Errorf("importing quote %d (%s): %w", i+1, q.ID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("importing quote %d (%s): %w", i+1, q.ID, err)
		}
		imported += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing import: %w", err)
	}
	return imported, nil
}

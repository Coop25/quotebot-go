package postgres

import "github.com/google/uuid"

func (a *postgresAccessor) AddQuote(quote string, guildID string) (Quote, error) {
	quoteUUID := uuid.New()
	_, err := a.db.Exec("INSERT INTO quotes (id, quote, guild_id) VALUES ($1, $2, $3)", quoteUUID, quote, guildID)
	if err != nil {
		return Quote{}, err
	}

	return Quote{
		ID:      quoteUUID,
		Quote:   quote,
		GuildID: guildID,
	}, nil
}

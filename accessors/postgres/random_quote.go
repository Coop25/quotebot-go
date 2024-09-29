package postgres

func (a *postgresAccessor) GetRandomQuote(guildID string) (Quote, error) {
	row := a.db.QueryRow("SELECT id, quote FROM quotes WHERE guild_id = $1 ORDER BY RANDOM() LIMIT 1", guildID)
	var quote Quote
	err := row.Scan(&quote.ID, &quote.Quote)
	if err != nil {
		return Quote{}, err
	}

	quote.GuildID = guildID
	return quote, nil
}

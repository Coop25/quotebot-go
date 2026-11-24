package postgres

import (
	"github.com/google/uuid"
)

type PostgresAccessor interface {
	GetRandomQuote(guildID string) (Quote, error)
	GetRandomQuotes(guildID string, limit int) ([]Quote, error)
	AddQuote(quote string, guildID string) (Quote, error)
	AllQuotes() ([]Quote, error)
	ImportQuotes(quotes []Quote) (imported int, err error)
}

type Quote struct {
	ID      uuid.UUID
	Quote   string
	GuildID string
}

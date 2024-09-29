package postgres

import (
	"github.com/google/uuid"
)

type PostgresAccessor interface {
	GetRandomQuote(guildID string) (Quote, error)
	AddQuote(quote string, guildID string) (Quote, error)
}

type Quote struct {
	ID      uuid.UUID
	Quote   string
	GuildID string
}

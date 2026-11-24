package commands

import (
	"errors"
	"log/slog"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

const (
	// maxMessageLength is Discord's limit on message content length.
	maxMessageLength = 2000
	// MaxQuoteLength leaves room for the quote ID header that is added when a
	// quote is shown, so a single quote always fits in one message.
	MaxQuoteLength = 1900
)

// truncate shortens s to at most max characters, marking the cut with an
// ellipsis. Quotes saved before MaxQuoteLength existed can be longer than
// Discord allows.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// noMentions stops user-submitted quote text from pinging users, roles or
// @everyone when the bot repeats it.
var noMentions = &discord.AllowedMentions{
	Parse: []discord.AllowedMentionType{},
	Roles: []snowflake.ID{},
	Users: []snowflake.ID{},
}

// messageCreator is implemented by both slash command and modal submit events.
type messageCreator interface {
	CreateMessage(messageCreate discord.MessageCreate, opts ...rest.RequestOpt) error
}

// replyError sends an ephemeral message so the user sees what went wrong
// instead of Discord's generic "This interaction failed".
func replyError(event messageCreator, content string) {
	err := event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(content).
		SetEphemeral(true).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending error response", slog.Any("err", err))
	}
}

// replyQuoteLookupError tells the user why a quote could not be fetched.
func replyQuoteLookupError(event messageCreator, err error) {
	if errors.Is(err, postgres.ErrNoQuotes) {
		replyError(event, "No quotes have been added yet. Use `/add-quote` to add one!")
		return
	}
	replyError(event, "Something went wrong while getting a quote. Please try again later.")
}

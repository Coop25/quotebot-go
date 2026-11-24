package commands

import (
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// maxMultiQuoteCount caps how many quotes one /multi-quote can ask for.
const maxMultiQuoteCount = 20

var MultiQuoteCommandCreate = discord.SlashCommandCreate{
	Name:        "multi-quote",
	Description: "Get multiple random quotes",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionInt{
			Name:        "count",
			Description: "The number of quotes to return",
			Required:    true,
			MinValue:    intPtr(1),
			MaxValue:    intPtr(maxMultiQuoteCount),
		},
		discord.ApplicationCommandOptionBool{
			Name:        "ephemeral",
			Description: "If the response should only be visible to you",
			Required:    false,
		},
		discord.ApplicationCommandOptionBool{
			Name:        "show-id",
			Description: "If the quote id should be shown",
			Required:    false,
		},
	},
}

func SendRandomMultiQuote(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) {
	data := event.SlashCommandInteractionData()
	ephemeral := data.Bool("ephemeral")
	showID := data.Bool("show-id")

	// Discord enforces the option's min/max, but clamp anyway in case the
	// registered command is out of date.
	count := min(max(data.Int("count"), 1), maxMultiQuoteCount)

	quotes, err := pg.GetRandomQuotes(data.GuildID().String(), count)
	if err != nil {
		slog.Error("error on getting random quotes", slog.Any("err", err))
		replyQuoteLookupError(event, err)
		return
	}

	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(formatMultiQuote(quotes, showID, maxMessageLength)).
		SetEphemeral(ephemeral).
		SetAllowedMentions(noMentions).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending response", slog.Any("err", err))
	}
}

// formatMultiQuote renders as many of quotes as fit within maxLen characters.
// With showID, the IDs of the included quotes are listed first. If not even
// the first quote fits, it is truncated rather than returning nothing.
func formatMultiQuote(quotes []postgres.Quote, showID bool, maxLen int) string {
	if len(quotes) == 0 {
		return ""
	}

	result := ""
	for n := 1; n <= len(quotes); n++ {
		candidate := renderMultiQuote(quotes[:n], showID)
		if utf8.RuneCountInString(candidate) > maxLen {
			break
		}
		result = candidate
	}
	if result == "" {
		result = truncate(renderMultiQuote(quotes[:1], showID), maxLen)
	}
	return result
}

func renderMultiQuote(quotes []postgres.Quote, showID bool) string {
	var b strings.Builder
	if showID {
		for _, q := range quotes {
			b.WriteString("> **Quote ID: **" + q.ID.String() + "\n")
		}
		b.WriteString("\n")
	}
	for _, q := range quotes {
		b.WriteString(q.Quote + "\n")
	}
	return b.String()
}

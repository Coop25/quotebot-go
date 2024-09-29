package commands

import (
	"log/slog"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var MultiQuoteCommandCreate = discord.SlashCommandCreate{
	Name:        "multi-quote",
	Description: "Get a random quote",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionBool{
			Name:        "ephemeral",
			Description: "If the response should only be visible to you",
			Required:    true,
		},
		discord.ApplicationCommandOptionInt{
			Name:        "count",
			Description: "The number of quotes to return",
			Required:    true,
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
	count := int(data.Int("count"))
	quotes, err := getQuotesUntilLimit(event, pg, count, showID)
	if err != nil {
		slog.Error("error on getting random quote", slog.Any("err", err))
		return
	}

	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(quotes).
		SetEphemeral(ephemeral).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending response", slog.Any("err", err))
	}
}

func getQuotesUntilLimit(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor, count int, isShowID bool) (string, error) {
	data := event.SlashCommandInteractionData()
	quotes := ""
	quoteIDs := ""
	isCombined := false
	slog.Info("ShowID: ", slog.Bool("showID", isShowID))
	for i := 0; i < count; i++ {
		message, err := pg.GetRandomQuote(data.GuildID().String())
		if err != nil {
			slog.Error("error on getting random quote", slog.Any("err", err))
			return "", err
		}
		quoteIDs += "> **Quote ID: **" + message.ID.String() + "\n"
		if isShowID {
			if len(quotes+quoteIDs+"\n\n"+message.Quote) < 2000 {
				quotes += message.Quote + "\n"
				if len(quotes+quoteIDs+"\n")+100 > 2000 {
					quotes = quoteIDs + "\n" + quotes
					slog.Info("Multi-Quote: ", slog.Int("QuotesLength", len(quotes)), slog.Int("QuoteCount", i))
					isCombined = true
					break
				}
			} else {
				quotes = quoteIDs + "\n" + quotes
				slog.Info("Multi-Quote: ", slog.Int("QuotesLength", len(quotes)), slog.Int("QuoteCount", i))
				isCombined = true
				break
			}
		} else {
			if len(quotes+"\n"+message.Quote) < 2000 {
				quotes += message.Quote + "\n"
			} else {
				slog.Info("Multi-Quote: ", slog.Int("QuotesLength", len(quotes)), slog.Int("QuoteCount", i))
				isCombined = true
				break
			}
		}
	}

	if !isCombined && isShowID {
		quotes = quoteIDs + "\n" + quotes
	}

	return quotes, nil
}

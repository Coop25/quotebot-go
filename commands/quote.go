package commands

import (
	"log/slog"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var QuoteCommandCreate = discord.SlashCommandCreate{
	Name:        "quote",
	Description: "Get a random quote",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionBool{
			Name:        "ephemeral",
			Description: "If the response should only be visible to you",
			Required:    true,
		},
		discord.ApplicationCommandOptionBool{
			Name:        "show-id",
			Description: "If the quote id should be shown",
			Required:    false,
		},
	},
}

func SendRandomQuote(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) {
	data := event.SlashCommandInteractionData()
	message, err := pg.GetRandomQuote(data.GuildID().String())
	if err != nil {
		slog.Error("error on getting random quote", slog.Any("err", err))
		return
	}
	ephemeral := data.Bool("ephemeral")
	showID := data.Bool("show-id")

	if showID {
		message.Quote = "> **Quote Id:** " + message.ID.String()+"\n\n" + message.Quote
	}

	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(message.Quote).
		SetEphemeral(ephemeral).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending response", slog.Any("err", err))
	}
}
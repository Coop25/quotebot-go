package commands

import (
	"log/slog"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var AddQuoteCommandCreate = discord.SlashCommandCreate{
	Name:        "add-quote",
	Description: "Opens the add a quote box",
	Options:     []discord.ApplicationCommandOption{},
}
var AddQuoteModalName = AddQuoteCommandCreate.Name + "-modal"

func SendAddQuote(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(AddQuoteModalName).
		SetTitle("Add a Quote").
		AddActionRow(
			discord.NewTextInput("quote", discord.TextInputStyleParagraph, "Quote").
				WithRequired(true).
				WithPlaceholder("Type your quote here..."),
		).
		Build()

	err := event.Client().Rest().CreateInteractionResponse(event.ID(), event.Token(), discord.InteractionResponse{
		Type: discord.InteractionResponseTypeModal,
		Data: discord.ModalCreate{
			CustomID:   modal.CustomID,
			Title:      modal.Title,
			Components: modal.Components,
		},
	})
	if err != nil {
		slog.Error("error on creating modal", slog.Any("err", err))
	}
}

func HandleAddQuoteModalSubmit(event *events.ModalSubmitInteractionCreate, pg postgres.PostgresAccessor) {
	quote := event.Data.Text("quote")

	if quote == "" {
		slog.Error("error on retrieving quote component")
		return
	}
	guildID := event.GuildID().String()

	message, err := pg.AddQuote(quote, guildID)
	if err != nil {
		slog.Error("error on adding quote", slog.Any("err", err))
		return
	}
	slog.Info("Added quote", slog.Any("quote", message.Quote))

	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("__**Quote added:**__ " + message.Quote).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending response", slog.Any("err", err))
	}
}
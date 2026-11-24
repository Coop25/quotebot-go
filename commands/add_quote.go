package commands

import (
	"log/slog"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/webhook"
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
				WithMaxLength(MaxQuoteLength).
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

// HandleAddQuoteModalSubmit saves a submitted quote. newQuoteWebhook may be nil
// when no webhook is configured.
func HandleAddQuoteModalSubmit(event *events.ModalSubmitInteractionCreate, pg postgres.PostgresAccessor, newQuoteWebhook webhook.Client) {
	quote := event.Data.Text("quote")

	if quote == "" {
		slog.Error("error on retrieving quote component")
		replyError(event, "Your quote was empty, so nothing was saved.")
		return
	}
	guildID := event.GuildID().String()

	message, err := pg.AddQuote(quote, guildID)
	if err != nil {
		slog.Error("error on adding quote", slog.Any("err", err))
		replyError(event, "Something went wrong while saving your quote. Please try again later.")
		return
	}
	slog.Info("Added quote", slog.Any("quote", message.Quote))

	if newQuoteWebhook != nil {
		_, err = newQuoteWebhook.CreateMessage(discord.WebhookMessageCreate{
			Content:         "> __**Quote ID:**__ " + message.ID.String() + "\n\n" + message.Quote,
			AllowedMentions: noMentions,
		})
		if err != nil {
			slog.Error("error on sending webhook", slog.Any("err", err))
		}
		_, err = newQuoteWebhook.CreateMessage(discord.WebhookMessageCreate{
			Content: "**. . . . . . . . . . . . . . . . . . . . . . . . . . . .**",
		})
		if err != nil {
			slog.Error("error on sending webhook", slog.Any("err", err))
		}
	}

	err = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("> __**Quote ID:**__ " + message.ID.String() + "\n\n" + message.Quote).
		SetAllowedMentions(noMentions).
		Build(),
	)
	if err != nil {
		slog.Error("error on sending response", slog.Any("err", err))
	}
}

package commands

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
)

// The transfer commands are only registered when ENABLE_TRANSFER_COMMANDS is
// on. Discord also hides them from members without Administrator, and the
// handlers refuse everyone except OWNER_ID.
var (
	ExportCommandCreate = discord.SlashCommandCreate{
		Name:                     "export",
		Description:              "Owner only: DM yourself a backup of every quote",
		DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
	}
	ImportCommandCreate = discord.SlashCommandCreate{
		Name:                     "import",
		Description:              "Owner only: load quotes from an export file",
		DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionAttachment{
				Name:        "file",
				Description: "A quotebot-export-*.json file made by /export",
				Required:    true,
			},
		},
	}
)

// downloadClient fetches /import attachments from Discord's CDN.
var downloadClient = &http.Client{Timeout: 30 * time.Second}

// isOwner reports whether userID is the configured owner. Nobody is the owner
// when OWNER_ID is not set. Surrounding whitespace in OWNER_ID is ignored.
func isOwner(userID snowflake.ID, ownerID string) bool {
	ownerID = strings.TrimSpace(ownerID)
	return ownerID != "" && userID.String() == ownerID
}

// NewExportHandler returns the /export handler for the given owner.
func NewExportHandler(ownerID string) func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error {
	return func(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) error {
		if !isOwner(event.User().ID, ownerID) {
			replyError(event, "Only the bot owner can use this command.")
			return nil
		}
		if err := event.DeferCreateMessage(true); err != nil {
			return fmt.Errorf("deferring export reply: %w", err)
		}

		quotes, err := pg.AllQuotes()
		if err != nil {
			slog.Error("error on reading quotes for export", slog.Any("err", err))
			editReply(event, "Something went wrong while reading the quotes. Please try again later.")
			return err
		}
		if len(quotes) == 0 {
			return editReply(event, "There are no quotes to export.")
		}

		now := time.Now()
		data, err := encodeExport(quotes, now)
		if err != nil {
			editReply(event, "Something went wrong while building the export.")
			return err
		}
		name := exportFileName(now)

		if err := dmExport(event, data, name, len(quotes)); err != nil {
			slog.Warn("could not DM export, attaching to reply instead", slog.Any("err", err))
			return editReply(event,
				fmt.Sprintf("Couldn't DM you, so here is the export (%d quotes). Save it now: this message disappears.", len(quotes)),
				discord.NewFile(name, "", bytes.NewReader(data)))
		}
		slog.Info("exported quotes", slog.Int("count", len(quotes)))
		return editReply(event, fmt.Sprintf("Sent %d quotes to your DMs.", len(quotes)))
	}
}

func dmExport(event *events.ApplicationCommandInteractionCreate, data []byte, name string, count int) error {
	rest := event.Client().Rest()
	channel, err := rest.CreateDMChannel(event.User().ID)
	if err != nil {
		return err
	}
	_, err = rest.CreateMessage(channel.ID(), discord.NewMessageCreateBuilder().
		SetContentf("Quote export: %d quotes", count).
		AddFile(name, "", bytes.NewReader(data)).
		Build())
	return err
}

// NewImportHandler returns the /import handler for the given owner.
func NewImportHandler(ownerID string) func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error {
	return func(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) error {
		if !isOwner(event.User().ID, ownerID) {
			replyError(event, "Only the bot owner can use this command.")
			return nil
		}
		if err := event.DeferCreateMessage(true); err != nil {
			return fmt.Errorf("deferring import reply: %w", err)
		}

		attachment := event.SlashCommandInteractionData().Attachment("file")
		if attachment.Size > maxImportBytes {
			editReply(event, fmt.Sprintf("That file is too large: the limit is %d MB.", maxImportBytes>>20))
			return fmt.Errorf("import file too large: %d bytes", attachment.Size)
		}

		quotes, err := downloadExport(attachment.URL)
		if err != nil {
			editReply(event, "That file isn't a valid export: "+err.Error())
			return err
		}

		imported, err := pg.ImportQuotes(quotes)
		if err != nil {
			slog.Error("error on importing quotes", slog.Any("err", err))
			editReply(event, "Something went wrong while saving the quotes, so nothing was imported.")
			return err
		}
		slog.Info("imported quotes", slog.Int("imported", imported), slog.Int("total", len(quotes)))
		return editReply(event, fmt.Sprintf("Imported %d new quotes, skipped %d that already existed.", imported, len(quotes)-imported))
	}
}

func downloadExport(url string) ([]postgres.Quote, error) {
	resp, err := downloadClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not download it (%v)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not download it (HTTP %d)", resp.StatusCode)
	}
	return decodeExport(resp.Body)
}

// editReply replaces the deferred ephemeral reply. Once an interaction has
// been deferred, CreateMessage no longer works, so every reply goes here.
func editReply(event *events.ApplicationCommandInteractionCreate, content string, files ...*discord.File) error {
	_, err := event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(),
		discord.NewMessageUpdateBuilder().
			SetContent(content).
			AddFiles(files...).
			SetAllowedMentions(noMentions).
			Build())
	if err != nil {
		slog.Error("error on editing reply", slog.Any("err", err))
	}
	return err
}

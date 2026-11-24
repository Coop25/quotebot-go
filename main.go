package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/Coop25/quotebot-go/commands"
	"github.com/Coop25/quotebot-go/config"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
)

// gatewayReadyTimeout bounds how long startup waits for Discord's Ready event.
const gatewayReadyTimeout = 60 * time.Second

// commandHandler runs a slash command. It returns an error when the command
// failed, which refunds the user's cooldown.
type commandHandler func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error

// buildCommands returns the handlers and the command list to register with
// Discord. Registration replaces the whole list, so commands left out here
// (like /export and /import when the switch is off) disappear from Discord.
func buildCommands(cfg config.Config) (map[string]commandHandler, []discord.ApplicationCommandCreate) {
	handlers := map[string]commandHandler{
		commands.QuoteCommandCreate.Name:      commands.SendRandomQuote,
		commands.AddQuoteCommandCreate.Name:   commands.SendAddQuote,
		commands.MultiQuoteCommandCreate.Name: commands.SendRandomMultiQuote,
		commands.RollCommandCreate.Name:       commands.RollCommand,
		// Add more commands here
	}
	creates := []discord.ApplicationCommandCreate{
		commands.AddQuoteCommandCreate,
		commands.QuoteCommandCreate,
		commands.MultiQuoteCommandCreate,
		commands.RollCommandCreate,
		// Add more commands here
	}

	if cfg.EnableTransferCommands {
		handlers[commands.ExportCommandCreate.Name] = commands.NewExportHandler(cfg.OwnerID)
		handlers[commands.ImportCommandCreate.Name] = commands.NewImportHandler(cfg.OwnerID)
		creates = append(creates, commands.ExportCommandCreate, commands.ImportCommandCreate)
	}
	return handlers, creates
}

var (
	modalHandlers = map[string]func(*events.ModalSubmitInteractionCreate, postgres.PostgresAccessor, webhook.Client){
		commands.AddQuoteModalName: commands.HandleAddQuoteModalSubmit,
		// Add more modals here
	}

	// Cooldown map to track the last execution time of commands for each user
	commandCooldowns   = make(map[string]time.Time)
	commandCooldownsMu sync.Mutex

	nonCooldownCommands = []string{
		commands.AddQuoteCommandCreate.Name,
		commands.RollCommandCreate.Name,
		commands.ExportCommandCreate.Name,
		commands.ImportCommandCreate.Name,
	}

	// Commands that work in any channel regardless of ALLOWED_CHANNELS.
	anyChannelCommands = []string{
		commands.ExportCommandCreate.Name,
		commands.ImportCommandCreate.Name,
	}
)

func main() {
	if err := run(); err != nil {
		slog.Error("bot stopped with an error", slog.Any("err", err))
		os.Exit(1)
	}
}

// run starts the bot and blocks until it receives a shutdown signal. Any
// error that leaves the bot unable to work is returned so main can exit
// with a non-zero status (letting Docker or systemd restart it).
func run() error {
	slog.Info("starting quotebot...")
	slog.Info("disgo version", slog.String("version", disgo.Version))
	config := config.LoadConfig()
	postgresAccessor := postgres.New(&config)

	commandHandlers, newCommands := buildCommands(config)
	if config.EnableTransferCommands {
		if config.OwnerID == "" {
			slog.Warn("ENABLE_TRANSFER_COMMANDS is on but OWNER_ID is not set; /export and /import will refuse everyone")
		} else {
			slog.Info("transfer commands /export and /import are enabled")
		}
	}

	var newQuoteWebhook webhook.Client
	if config.NewQuoteWebhook != "" {
		var err error
		newQuoteWebhook, err = webhook.NewWithURL(config.NewQuoteWebhook)
		if err != nil {
			return fmt.Errorf("creating new quote webhook client: %w", err)
		}
		defer newQuoteWebhook.Close(context.TODO())
	}

	ready := make(chan struct{})
	var readyOnce sync.Once

	client, err := disgo.New(config.Token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(func(event bot.Event) {
			if _, ok := event.(*events.Ready); ok {
				readyOnce.Do(func() { close(ready) })
			}
			if e, ok := event.(*events.ApplicationCommandInteractionCreate); ok {
				commandListener(e, postgresAccessor, config, commandHandlers)
			}
			if e, ok := event.(*events.ModalSubmitInteractionCreate); ok {
				modalListener(e, postgresAccessor, config, newQuoteWebhook)
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("building disgo instance: %w", err)
	}

	guildID, err := snowflake.Parse(config.GuildID)
	if err != nil {
		return fmt.Errorf("parsing GUILD_ID: %w", err)
	}

	defer client.Close(context.TODO())

	if _, err = client.Rest().SetGuildCommands(client.ApplicationID(), guildID, newCommands); err != nil {
		// A 401 means the token is invalid, so the bot can never work.
		var restErr rest.Error // disgo returns rest.Error by value
		if errors.As(err, &restErr) && restErr.Response != nil && restErr.Response.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("registering commands: invalid TOKEN: %w", err)
		}
		// Otherwise not fatal: commands registered on a previous run keep working.
		slog.Error("error while registering commands", slog.Any("err", err))
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	// disgo's OpenGateway retries forever, and a rejected token only shows up
	// as the connection being closed later. Bound both and treat "never
	// became ready" as a startup failure.
	openCtx, cancel := context.WithTimeout(context.Background(), gatewayReadyTimeout)
	defer cancel()
	if err = client.OpenGateway(openCtx); err != nil {
		return fmt.Errorf("connecting to gateway: %w", err)
	}
	select {
	case <-ready:
	case <-openCtx.Done():
		return fmt.Errorf("gateway did not become ready within %s; check TOKEN and network access to Discord", gatewayReadyTimeout)
	case <-s:
		return nil
	}

	slog.Info("Bot is running. Press CTRL+C to exit.")
	<-s
	return nil
}

func commandListener(event *events.ApplicationCommandInteractionCreate, db postgres.PostgresAccessor, config config.Config, commandHandlers map[string]commandHandler) {
	if config.GuildID != event.GuildID().String() {
		return
	}

	data := event.SlashCommandInteractionData()
	ephemeral := data.Bool("ephemeral")
	configChannel := config.AllowedChannelsMap[data.CommandName()]
	isAllowed := true
	if configChannel != "" && !contains(anyChannelCommands, data.CommandName()) {
		isAllowed = event.Channel().ID().String() == configChannel
	}
	if !ephemeral && !isAllowed {
		event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("This command cannot be used in this channel.").
			SetEphemeral(true).
			Build(),
		)
		return
	}

	// Set when this use started a cooldown, so it can be refunded on failure.
	cooldownKey := ""
	if !ephemeral && !contains(nonCooldownCommands, data.CommandName()) {
		key := event.User().ID.String() + data.CommandName()
		if remaining, onCooldown := checkCooldown(key, config.CooldownDuration); onCooldown {
			event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContent("You are on cooldown. Please wait " + remaining.Round(time.Second).String() + " before using this command again.").
				SetEphemeral(true).
				Build(),
			)
			return
		}
		cooldownKey = key
	}

	handler, ok := commandHandlers[data.CommandName()]
	if !ok {
		slog.Warn("unknown command", slog.String("command", data.CommandName()))
		return
	}
	// Don't make users wait out a cooldown for a command that failed
	// (e.g. "no quotes yet" or a database error), so they can retry.
	if err := handler(event, db); err != nil && cooldownKey != "" {
		clearCooldown(cooldownKey)
	}
}

func modalListener(event *events.ModalSubmitInteractionCreate, db postgres.PostgresAccessor, config config.Config, newQuoteWebhook webhook.Client) {
	if config.GuildID != event.GuildID().String() {
		return
	}

	if handler, ok := modalHandlers[event.Data.CustomID]; ok {
		handler(event, db, newQuoteWebhook)
	} else {
		slog.Warn("unknown modal", slog.String("modal", event.Data.CustomID))
	}
}

// checkCooldown reports whether key is still on cooldown and, if it is not,
// starts a new cooldown period for it.
func checkCooldown(key string, duration time.Duration) (time.Duration, bool) {
	commandCooldownsMu.Lock()
	defer commandCooldownsMu.Unlock()

	if lastExecution, ok := commandCooldowns[key]; ok {
		if elapsed := time.Since(lastExecution); elapsed < duration {
			return duration - elapsed, true
		}
	}
	commandCooldowns[key] = time.Now()
	return 0, false
}

// clearCooldown removes any cooldown recorded for key.
func clearCooldown(key string) {
	commandCooldownsMu.Lock()
	defer commandCooldownsMu.Unlock()
	delete(commandCooldowns, key)
}

// contains checks if a slice contains a specific string
func contains(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

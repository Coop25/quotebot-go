package main

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
)

var (
	commandHandlers = map[string]func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor){
		commands.QuoteCommandCreate.Name:      commands.SendRandomQuote,
		commands.AddQuoteCommandCreate.Name:   commands.SendAddQuote,
		commands.MultiQuoteCommandCreate.Name: commands.SendRandomMultiQuote,
		commands.RollCommandCreate.Name:       commands.RollCommand,
		// Add more commands here
	}

	modalHandlers = map[string]func(*events.ModalSubmitInteractionCreate, postgres.PostgresAccessor, webhook.Client){
		commands.AddQuoteModalName: commands.HandleAddQuoteModalSubmit,
		// Add more modals here
	}

	newCommands = []discord.ApplicationCommandCreate{
		commands.AddQuoteCommandCreate,
		commands.QuoteCommandCreate,
		commands.MultiQuoteCommandCreate,
		commands.RollCommandCreate,
		// Add more commands here
	}

	// Cooldown map to track the last execution time of commands for each user
	commandCooldowns   = make(map[string]time.Time)
	commandCooldownsMu sync.Mutex

	nonCooldownCommands = []string{
		commands.AddQuoteCommandCreate.Name,
		commands.RollCommandCreate.Name,
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

	var newQuoteWebhook webhook.Client
	if config.NewQuoteWebhook != "" {
		var err error
		newQuoteWebhook, err = webhook.NewWithURL(config.NewQuoteWebhook)
		if err != nil {
			return fmt.Errorf("creating new quote webhook client: %w", err)
		}
		defer newQuoteWebhook.Close(context.TODO())
	}

	client, err := disgo.New(config.Token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(func(event bot.Event) {
			if e, ok := event.(*events.ApplicationCommandInteractionCreate); ok {
				commandListener(e, postgresAccessor, config)
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

	// Not fatal: commands registered on a previous run keep working.
	if _, err = client.Rest().SetGuildCommands(client.ApplicationID(), guildID, newCommands); err != nil {
		slog.Error("error while registering commands", slog.Any("err", err))
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		return fmt.Errorf("connecting to gateway: %w", err)
	}

	slog.Info("Bot is running. Press CTRL+C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
	return nil
}

func commandListener(event *events.ApplicationCommandInteractionCreate, db postgres.PostgresAccessor, config config.Config) {
	if config.GuildID != event.GuildID().String() {
		return
	}

	data := event.SlashCommandInteractionData()
	ephemeral := data.Bool("ephemeral")
	configChannel := config.AllowedChannelsMap[data.CommandName()]
	isAllowed := true
	if configChannel != "" {
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
	}

	if handler, ok := commandHandlers[data.CommandName()]; ok {
		handler(event, db)
	} else {
		slog.Warn("unknown command", slog.String("command", data.CommandName()))
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

// contains checks if a slice contains a specific string
func contains(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

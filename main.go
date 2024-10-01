package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/Coop25/quotebot-go/commands"
	"github.com/Coop25/quotebot-go/config"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

var (
	commandHandlers = map[string]func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor){
		commands.QuoteCommandCreate.Name:      commands.SendRandomQuote,
		commands.AddQuoteCommandCreate.Name:   commands.SendAddQuote,
		commands.MultiQuoteCommandCreate.Name: commands.SendRandomMultiQuote,
		// Add more commands here
	}

	modalHandlers = map[string]func(*events.ModalSubmitInteractionCreate, postgres.PostgresAccessor, config.Config){
		commands.AddQuoteModalName: commands.HandleAddQuoteModalSubmit,
		// Add more modals here
	}

	newCommands = []discord.ApplicationCommandCreate{
		commands.AddQuoteCommandCreate,
		commands.QuoteCommandCreate,
		commands.MultiQuoteCommandCreate,
		// Add more commands here
	}

	// Cooldown map to track the last execution time of commands for each user
	commandCooldowns = make(map[string]time.Time)
)

func main() {
	slog.Info("starting example...")
	slog.Info("disgo version", slog.String("version", disgo.Version))
	config := config.LoadConfig()
	postgresAccessor := postgres.New(&config)

	client, err := disgo.New(config.Token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(func(event bot.Event) {
			if e, ok := event.(*events.ApplicationCommandInteractionCreate); ok {
				commandListener(e, postgresAccessor, config)
			}
			if e, ok := event.(*events.ModalSubmitInteractionCreate); ok {
				modalListener(e, postgresAccessor, config)
			}
		}),
	)
	if err != nil {
		slog.Error("error while building disgo instance", slog.Any("err", err))
		return
	}

	guildID, err := snowflake.Parse(config.GuildID)
	if err != nil {
		slog.Error("error while parsing guildID", slog.Any("err", err))
		return
	}

	defer client.Close(context.TODO())

	if _, err = client.Rest().SetGuildCommands(client.ApplicationID(), guildID, newCommands); err != nil {
		slog.Error("error while registering commands", slog.Any("err", err))
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		slog.Error("error while connecting to gateway", slog.Any("err", err))
	}

	slog.Info("Bot is running. Press CTRL+C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}

func commandListener(event *events.ApplicationCommandInteractionCreate, db postgres.PostgresAccessor, config config.Config) {
	if config.GuildID != event.GuildID().String() {
		return
	}

	data := event.SlashCommandInteractionData()
	ephemeral := data.Bool("ephemeral")
	if !ephemeral && !isAllowedChannel(data.CommandName()+":"+event.Channel().ID().String(), config) {
		event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("This command cannot be used in this channel.").
			SetEphemeral(true).
			Build(),
		)
		return
	}

	if !ephemeral && data.CommandName() != commands.AddQuoteCommandCreate.Name {
		userID := event.User().ID.String()
		commandName := event.SlashCommandInteractionData().CommandName()

		// Check if the user is within the cooldown period
		if lastExecution, ok := commandCooldowns[userID+commandName]; ok {
			if time.Since(lastExecution) < config.CooldownDuration {
				remaining := config.CooldownDuration - time.Since(lastExecution)
				event.CreateMessage(discord.NewMessageCreateBuilder().
					SetContent("You are on cooldown. Please wait " + remaining.String() + " before using this command again.").
					SetEphemeral(true).
					Build(),
				)
				return
			}
		}

		// Update the last execution time
		commandCooldowns[userID+commandName] = time.Now()
	}

	if handler, ok := commandHandlers[data.CommandName()]; ok {
		handler(event, db)
	} else {
		slog.Warn("unknown command", slog.String("command", data.CommandName()))
	}
}

func modalListener(event *events.ModalSubmitInteractionCreate, db postgres.PostgresAccessor, config config.Config) {
	if config.GuildID != event.GuildID().String() {
		return
	}

	if handler, ok := modalHandlers[event.Data.CustomID]; ok {
		handler(event, db, config)
	} else {
		slog.Warn("unknown modal", slog.String("modal", event.Data.CustomID))
	}
}

func isAllowedChannel(commandChannel string, config config.Config) bool {
	if config.AllowedChannels == "" {
		return true
	}
	parts := strings.Split(config.AllowedChannels, ",")
	doesContainCmdName := false
	for _, allowedChannel := range parts {
		if commandChannel == allowedChannel {
			return true
		}

		if strings.Split(commandChannel, ":")[0] == strings.Split(allowedChannel, ":")[0] {
			doesContainCmdName = true
		}
	}

	return !doesContainCmdName
}

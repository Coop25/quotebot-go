package commands

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var RollCommandCreate = discord.SlashCommandCreate{
	Name:        "roll",
	Description: "Roll a die with a specified number of sides",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionInt{
			Name:        "sides",
			Description: "Number of sides on the die",
			Required:    true,
			MinValue:    intPtr(2),
		},
		discord.ApplicationCommandOptionBool{
			Name:        "ephemeral",
			Description: "If the response should only be visible to you",
			Required:    false,
		},
	},
}

func intPtr(i int) *int {
	return &i
}

func RollCommand(event *events.ApplicationCommandInteractionCreate, pg postgres.PostgresAccessor) {
	data := event.SlashCommandInteractionData()
	sides := data.Int("sides")
	ephemeral := data.Bool("ephemeral")

	// Create a new random number generator
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	roll := rng.Intn(sides) + 1

	// Generate the ASCII art for the roll result
	art := generateDiceArt(roll, sides)

	response := discord.NewMessageCreateBuilder().
		SetContent(fmt.Sprintf("You rolled a %d on a %d-sided die.\n```\n%s```", roll, sides, art)).
		SetEphemeral(ephemeral).
		Build()

	if err := event.CreateMessage(response); err != nil {
		slog.Error("error sending response", slog.Any("err", err))
	}
}

func generateDiceArt(roll, sides int) string {
    rollStr := strconv.Itoa(roll)
    sidesStr := strconv.Itoa(sides)
    width := len(sidesStr)
    if len(rollStr) > width {
        width = len(rollStr)
    }

    topBottom := "+" + strings.Repeat("-", width+2) + "+"
    middle := "|" + strings.Repeat(" ", width+2) + "|"

    // Center the roll number
    padding := (width - len(rollStr)) / 2
    rollLine := "|" + strings.Repeat(" ", padding+1) + rollStr + strings.Repeat(" ", width-len(rollStr)-padding+1) + "|"

    return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", topBottom, middle, rollLine, middle, topBottom)
}

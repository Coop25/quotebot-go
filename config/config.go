package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Token              string        `envconfig:"TOKEN" required:"true"`
	PGDBHost           string        `envconfig:"PG_DB_HOST" default:"localhost"`
	PGDBPort           string        `envconfig:"PG_DB_PORT" default:"5432"`
	PGDBUser           string        `envconfig:"PG_DB_USER" default:"postgres"`
	PGDBPass           string        `envconfig:"PG_DB_PASS" default:"password"`
	PGDBName           string        `envconfig:"PG_DB_NAME" default:"memeindex"`
	PGDBSSLMode        string        `envconfig:"PG_DB_SSL_MODE" default:"disable"`
	GuildID            string        `envconfig:"GUILD_ID" required:"true"`
	NewQuoteWebhook    string        `envconfig:"NEW_QUOTE_WEBHOOK" default:""`
	AllowedChannels    string        `envconfig:"ALLOWED_CHANNELS" default:""`
	CooldownDuration   time.Duration `envconfig:"COOLDOWN_DURATION" default:"60s"`
	AllowedChannelsMap map[string]string
}

func LoadConfig() Config {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	config.AllowedChannelsMap, err = parseAllowedChannels(config.AllowedChannels)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	return config
}

// parseAllowedChannels parses a comma separated list of command:channelID
// pairs, e.g. "quote:123,multi-quote:456".
func parseAllowedChannels(value string) (map[string]string, error) {
	channels := make(map[string]string)
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		command, channelID, ok := strings.Cut(entry, ":")
		command, channelID = strings.TrimSpace(command), strings.TrimSpace(channelID)
		if !ok || command == "" || channelID == "" {
			return nil, fmt.Errorf("invalid ALLOWED_CHANNELS entry %q: expected command:channelID", entry)
		}
		channels[command] = channelID
	}
	return channels, nil
}

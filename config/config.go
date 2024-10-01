package config

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Token            string        `envconfig:"TOKEN" default:"DISCORD_BOT_TOKEN"`
	PGDBHost         string        `envconfig:"PG_DB_HOST" default:"localhost"`
	PGDBPort         string        `envconfig:"PG_DB_PORT" default:"5432"`
	PGDBUser         string        `envconfig:"PG_DB_USER" default:"postgres"`
	PGDBPass         string        `envconfig:"PG_DB_PASS" default:"password"`
	PGDBName         string        `envconfig:"PG_DB_NAME" default:"memeindex"`
	PGDBSSLMode      string        `envconfig:"PG_DB_SSL_MODE" default:"disable"`
	GuildID          string        `envconfig:"GUILD_ID" default:"123456789123456789"`
	NewQuoteWebhook  string        `envconfig:"NEW_QUOTE_WEBHOOK" default:""`
	AllowedChannels  string        `envconfig:"ALLOWED_CHANNELS" default:""`
	CooldownDuration time.Duration `envconfig:"COOLDOWN_DURATION" default:"60s"`
}

func LoadConfig() Config {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}
	return config
}

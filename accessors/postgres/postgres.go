package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Coop25/quotebot-go/config"
	_ "github.com/lib/pq"
)

type postgresAccessor struct {
	config *config.Config
	db     *sql.DB
}

func New(config *config.Config) *postgresAccessor {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.PGDBHost, config.PGDBPort, config.PGDBUser, config.PGDBPass, config.PGDBName, config.PGDBSSLMode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// sql.Open only validates the DSN; ping so bad credentials or an
	// unreachable host fail at startup rather than on the first command.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	return &postgresAccessor{
		config: config,
		db:     db,
	}
}

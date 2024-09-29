package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func runMigrations(db *sql.DB, migrationsDir string) error {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			migrationPath := filepath.Join(migrationsDir, file.Name())
			migration, err := os.ReadFile(migrationPath)
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
			}

			_, err = db.Exec(string(migration))
			if err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
			}

			log.Printf("Successfully executed migration: %s", file.Name())
		}
	}

	return nil
}

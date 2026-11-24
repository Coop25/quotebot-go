package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Coop25/quotebot-go/accessors/postgres"
	"github.com/google/uuid"
)

const (
	// exportFormatVersion is bumped if the export file layout ever changes,
	// so an older bot refuses a file it does not understand.
	exportFormatVersion = 1
	// maxImportBytes is the largest export file /import will accept.
	maxImportBytes = 25 << 20
)

type exportFile struct {
	Version    int           `json:"version"`
	ExportedAt string        `json:"exported_at"`
	Quotes     []exportQuote `json:"quotes"`
}

type exportQuote struct {
	ID      string `json:"id"`
	GuildID string `json:"guild_id"`
	Quote   string `json:"quote"`
}

// encodeExport renders quotes as an export file.
func encodeExport(quotes []postgres.Quote, now time.Time) ([]byte, error) {
	file := exportFile{
		Version:    exportFormatVersion,
		ExportedAt: now.UTC().Format(time.RFC3339),
		Quotes:     make([]exportQuote, len(quotes)),
	}
	for i, q := range quotes {
		file.Quotes[i] = exportQuote{ID: q.ID.String(), GuildID: q.GuildID, Quote: q.Quote}
	}
	return json.MarshalIndent(file, "", "  ")
}

// decodeExport parses and validates an export file. Its errors are shown to
// the user, so they say what is wrong in plain terms.
func decodeExport(r io.Reader) ([]postgres.Quote, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("could not read the file: %w", err)
	}
	if len(data) > maxImportBytes {
		return nil, fmt.Errorf("the file is larger than %d MB", maxImportBytes>>20)
	}

	var file exportFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("it is not valid JSON (%v)", err)
	}
	if file.Version != exportFormatVersion {
		return nil, fmt.Errorf("expected export format version %d, got %d", exportFormatVersion, file.Version)
	}

	quotes := make([]postgres.Quote, len(file.Quotes))
	for i, q := range file.Quotes {
		id, err := uuid.Parse(q.ID)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("quote %d has an invalid id %q", i+1, q.ID)
		}
		if q.GuildID == "" {
			return nil, fmt.Errorf("quote %d has no guild_id", i+1)
		}
		if q.Quote == "" {
			return nil, fmt.Errorf("quote %d has no text", i+1)
		}
		quotes[i] = postgres.Quote{ID: id, GuildID: q.GuildID, Quote: q.Quote}
	}
	return quotes, nil
}

// exportFileName is the name of the export file for the given day.
func exportFileName(now time.Time) string {
	return "quotebot-export-" + now.UTC().Format("2006-01-02") + ".json"
}

# Quote Export / Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add owner-only `/export` and `/import` commands, switched off by default, plus a docker-compose stack with a bundled Postgres, so the bot's quotes can be moved off a machine that is reachable only through Portainer.

**Architecture:**
- The export file format lives in its own file in `commands`, as pure encode/decode functions.
- Two new accessor methods read all quotes and bulk-insert them in one transaction.
- `main.go` builds the command list at startup and adds the two transfer commands only when `ENABLE_TRANSFER_COMMANDS` is true. Because `SetGuildCommands` replaces the whole list, the commands vanish from Discord when the switch is off.

**Tech Stack:** Go 1.21, disgo v0.18.12, lib/pq, Postgres 16, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-11-24-export-import-design.md`

## Global Constraints

- Go 1.21 language level (CI uses `go-version-file: go.mod`). No newer stdlib APIs.
- No new module dependencies.
- New env vars:
  - `OWNER_ID` (string, no default).
  - `ENABLE_TRANSFER_COMMANDS` (bool, default `false`).
- Export file name: `quotebot-export-YYYY-MM-DD.json`.
- Export file shape: `{"version": 1, "exported_at": "<RFC 3339>", "quotes": [{"id", "guild_id", "quote"}]}`.
- Import refuses attachments over 25 MB (`25 << 20` bytes).
- Import uses one transaction with `ON CONFLICT (id) DO NOTHING`. Any error writes nothing.
- Both commands:
  - refuse anyone whose user ID is not `OWNER_ID`, and refuse everyone when `OWNER_ID` is empty;
  - skip cooldowns and channel restrictions;
  - set `DefaultMemberPermissions` to Administrator;
  - reply ephemerally and defer before doing work.
- All Go files pass `gofmt -l` (CI enforces).

## Review Focus

1. **Someone other than the owner runs `/export` or `/import`, admins included:** refused with an ephemeral message, and nothing is read or written. Covered by `TestIsOwner` in Task 3.
2. **`ENABLE_TRANSFER_COMMANDS=true` but `OWNER_ID` unset:** both commands refuse everyone. Covered by `TestIsOwner` (empty owner) in Task 3.
3. **The same file is imported twice:** the second run imports 0 and skips N. Covered by `TestImportQuotesSkipsExisting` in Task 2.
4. **The wrong file is attached (an image, other JSON, a truncated export, a future version):** a clear error, and nothing is written. Covered by `TestDecodeExportRejects` in Task 1.
5. **A database error partway through an import:** earlier rows in the batch are rolled back. Covered by `TestImportQuotesIsAllOrNothing` in Task 2.

---

### Task 1: Export file format

**Files:**
- Create: `commands/transfer_file.go`
- Test: `commands/transfer_file_test.go`

**Interfaces:**
- Produces:
  - `const exportFormatVersion = 1`
  - `const maxImportBytes = 25 << 20`
  - `func encodeExport(quotes []postgres.Quote, now time.Time) ([]byte, error)`: indented JSON in the Global Constraints shape. `exported_at` is `now.UTC()` in RFC 3339.
  - `func decodeExport(r io.Reader) ([]postgres.Quote, error)`: reads at most `maxImportBytes + 1` bytes and errors if over. Rejects non-JSON, `version != 1`, a missing or nil `id`, an empty `guild_id`, and an empty `quote`. Errors are user-readable and name the bad entry's index.
  - `func exportFileName(now time.Time) string`: returns `quotebot-export-2026-11-24.json`.

- [ ] **Step 1: Write the failing tests**
  - `TestExportRoundTrip`: encode 3 quotes with fixed time `2026-11-24T10:00:00Z`, decode, and `reflect.DeepEqual` the quotes. Also assert the JSON contains `"version": 1` and `"exported_at": "2026-11-24T10:00:00Z"`.
  - `TestExportFileName`: equals `quotebot-export-2026-11-24.json`.
  - `TestDecodeExportRejects`: table cases, each expecting a non-nil error:
    - `"not json"`
    - `{"version":2,"quotes":[]}`
    - `{"version":1,"quotes":[{"id":"nope","guild_id":"1","quote":"q"}]}`
    - a nil UUID id
    - an empty guild_id
    - an empty quote
    - a reader of `maxImportBytes+1` bytes of spaces
  - `{"version":1,"quotes":[]}` decodes to an empty slice with no error.
- [ ] **Step 2: Run** `go test ./commands -run 'Export|Decode'`. Expected: FAIL (undefined functions).
- [ ] **Step 3: Implement** the four symbols above using unexported structs with json tags `version`, `exported_at`, `quotes`, `id`, `guild_id`, `quote`.
- [ ] **Step 4: Run** `go test ./commands`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add quote export file format`.

### Task 2: Accessor methods for all quotes and bulk import

**Files:**
- Create: `accessors/postgres/transfer.go`
- Modify: `accessors/postgres/models.go` (the `PostgresAccessor` interface)
- Test: `accessors/postgres/transfer_test.go` (reuses `newTestAccessor` and `newGuildID` from `postgres_test.go`, and skips without `TEST_DATABASE_URL`)

**Interfaces:**
- Produces:
  - `AllQuotes() ([]Quote, error)`: every quote across all guilds, `ORDER BY guild_id, id`. Returns an empty slice, not an error, when there are none.
  - `ImportQuotes(quotes []Quote) (imported int, err error)`: one transaction. Each row is `INSERT ... ON CONFLICT (id) DO NOTHING`, and `imported` is the sum of `RowsAffected`. On any error it rolls back and returns `0, err`.

- [ ] **Step 1: Write the failing tests.**
  - `TestAllQuotesIncludesEveryGuild`: add one quote to each of two new guilds. Both IDs appear in `AllQuotes()` with the right guild and text.
  - `TestImportQuotesSkipsExisting`: build 3 quotes with fresh UUIDs in a new guild.
    - First import returns 3.
    - Second import returns 0.
    - `GetRandomQuotes(guild, 10)` returns exactly 3.
  - `TestImportQuotesIsAllOrNothing`: import `[good, bad]`, where `bad.Quote` contains `"\x00"` (Postgres rejects NUL in text). This returns an error and `0`, and `GetRandomQuotes(guild, 10)` returns `ErrNoQuotes`.
- [ ] **Step 2: Run** `go vet ./accessors/postgres`. Expected: FAIL (methods undefined).
- [ ] **Step 3: Implement** both methods and add them to the interface.
- [ ] **Step 4: Run** `go test ./...`. Expected: PASS locally with the DB tests skipped. Real verification happens in CI (Task 5).
- [ ] **Step 5: Commit** `feat: add AllQuotes and transactional ImportQuotes`.

### Task 3: `/export` and `/import` commands, owner check, on/off switch

**Files:**
- Create: `commands/transfer.go`
- Test: `commands/transfer_test.go`, `main_test.go`
- Modify:
  - `config/config.go`: add `OwnerID string` (`envconfig:"OWNER_ID"`) and `EnableTransferCommands bool` (`envconfig:"ENABLE_TRANSFER_COMMANDS" default:"false"`).
  - `main.go`: replace the static `commandHandlers`/`newCommands` vars with a builder.

**Interfaces:**
- Consumes: Task 1 `encodeExport`, `decodeExport`, `exportFileName`, `maxImportBytes`. Task 2 `AllQuotes`, `ImportQuotes`.
- Produces:
  - `var ExportCommandCreate`: name `export`, description `Owner only: DM yourself a backup of every quote`.
  - `var ImportCommandCreate`: name `import`, description `Owner only: load quotes from an export file`, with a required `ApplicationCommandOptionAttachment` named `file`.
  - Both set `DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionAdministrator)` (`github.com/disgoorg/json`).
  - `func isOwner(userID snowflake.ID, ownerID string) bool`: false when `ownerID` is empty.
  - `func NewExportHandler(ownerID string) func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error`
  - `func NewImportHandler(ownerID string) func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error`
  - In `main.go`: `func buildCommands(cfg config.Config) (map[string]commandHandler, []discord.ApplicationCommandCreate)`, where `type commandHandler func(*events.ApplicationCommandInteractionCreate, postgres.PostgresAccessor) error`.

**Handler behavior:**
- **Both handlers:**
  1. If the user is not the owner, `replyError` "Only the bot owner can use this command." and return nil (no cooldown to refund).
  2. Otherwise `event.DeferCreateMessage(true)`.
  3. Finish by editing the deferred reply with `event.Client().Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), ...)`. After deferring, every reply, errors included, must go through this call: `replyError`/`CreateMessage` fail once an interaction has been deferred.
  4. Return a non-nil error for any failure after the owner check.
- **Export:**
  - No quotes: the reply is "There are no quotes to export." and nothing is sent.
  - Otherwise, `CreateDMChannel(owner)` and post the file with content "Quote export: N quotes". The reply is "Sent N quotes to your DMs."
  - If the DM fails, attach the file to the deferred reply instead, with "Couldn't DM you, so here is the export (N quotes). Save it now: this message disappears."
- **Import:**
  - Reject `attachment.Size > maxImportBytes` before downloading.
  - Download `attachment.URL` with an `http.Client{Timeout: 30 * time.Second}`, then `decodeExport`, then `ImportQuotes`.
  - The reply is "Imported N new quotes, skipped M that already existed." where M = total − N.
  - Decode errors are shown to the user verbatim, prefixed with "That file isn't a valid export: ".
- **`main.go`:**
  - The transfer commands are added to `buildCommands` only when `cfg.EnableTransferCommands`.
  - `export` and `import` join `nonCooldownCommands`. `commandListener` skips the channel restriction check for them.
  - On startup, log a warning if the switch is on but `OWNER_ID` is empty.

- [ ] **Step 1: Write the failing tests.**
  - `TestIsOwner`: matching ID gives true, a different ID gives false, and an empty owner with any ID gives false.
  - `TestBuildCommandsTransferSwitch` in `main_test.go`:
    - with `EnableTransferCommands: false`, neither `export` nor `import` is in the map or the slice, and `quote` is;
    - with `true`, both are in both.
- [ ] **Step 2: Run** `go test ./... -run 'IsOwner|BuildCommands'`. Expected: FAIL.
- [ ] **Step 3: Implement** the config fields, `commands/transfer.go`, and the `main.go` builder and listener changes.
- [ ] **Step 4: Run** `go vet ./... && go test ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat: add owner-only /export and /import behind ENABLE_TRANSFER_COMMANDS`.

### Task 4: Docker Compose stack and docs

**Files:**
- Create: `docker-compose.yml`
- Modify: `.env.sample`, `README.md`

**Compose file:**
- **`db` service:**
  - image `postgres:16`;
  - `POSTGRES_USER=quotebot`, `POSTGRES_DB=quotebot`, `POSTGRES_PASSWORD=${PG_DB_PASS:-quotebot}`;
  - named volume `quotebot-db` at `/var/lib/postgresql/data`;
  - healthcheck `pg_isready -U quotebot -d quotebot`;
  - **no published ports**, so it is only reachable inside the stack;
  - `restart: unless-stopped`.
- **`bot` service:**
  - image `ghcr.io/coop25/discord-go:latest`;
  - `depends_on: db: condition: service_healthy`;
  - `restart: unless-stopped`;
  - environment `PG_DB_HOST=db`, `PG_DB_PORT=5432`, `PG_DB_USER=quotebot`, `PG_DB_PASS=${PG_DB_PASS:-quotebot}`, `PG_DB_NAME=quotebot`, `PG_DB_SSL_MODE=disable`;
  - `TOKEN`, `GUILD_ID`, `OWNER_ID`, `NEW_QUOTE_WEBHOOK`, `ALLOWED_CHANNELS`, `COOLDOWN_DURATION` and `ENABLE_TRANSFER_COMMANDS` are passed through with `${VAR}` / `${VAR:-}` (and `${ENABLE_TRANSFER_COMMANDS:-false}`), so they come from a `.env` file or Portainer's stack environment.

**Docs:**
- `.env.sample` gains `OWNER_ID=` and `ENABLE_TRANSFER_COMMANDS=false`.
- A README section "Running with Docker Compose / Portainer".
- A README section "Moving the bot to a new host" with the six migration steps from the spec. It must point out that a GitHub release has to be published first so `:latest` contains the commands.

- [ ] **Step 1: Write** the three files.
- [ ] **Step 2: Run** `TOKEN=x GUILD_ID=1 docker compose config -q`. Expected: exit 0, no output (works without the Docker daemon).
- [ ] **Step 3: Commit** `feat: add docker-compose stack with bundled Postgres; document migration`.

### Task 5: Ship

- [ ] **Step 1:** `gofmt -l` on all Go files gives no output, and `go vet ./...` and `go test ./...` pass.
- [ ] **Step 2:** Push the branch, open a PR to `master`, and wait for the CI `test` and `docker` checks. Confirm the CI log shows `TestImportQuotesSkipsExisting`, `TestImportQuotesIsAllOrNothing` and `TestAllQuotesIncludesEveryGuild` as PASS, not SKIP.
- [ ] **Step 3:** Merge only after the checks pass and the owner approves.

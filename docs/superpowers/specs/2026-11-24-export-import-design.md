# Quote export / import commands

## Why

The bot runs on a machine that is being retired. The only access to it is
Portainer, which can update the bot's container but cannot reach the
Postgres database. There is no backup. The quotes have to come out through
the bot itself and go into a new instance somewhere else.

## Goals

- `/export`: the bot DMs the owner a file containing every quote.
- `/import`: the owner attaches that file to `/import` on a new instance and
  the quotes are loaded.
- Only the owner's Discord account can run either command.
- `/export` doubles as a manual backup that can be run at any time.
- A `docker-compose.yml` that runs the bot and its own Postgres in one stack
  (deployable as a Portainer stack). The bot is preconfigured to reach that
  database, so only the Discord settings (`TOKEN`, `GUILD_ID`, `OWNER_ID`,
  and so on) have to be filled in.

## Non-goals

- Automatic import on startup and scheduled backups (possible later).
- Moving quotes to a different Discord server: quotes keep the guild ID they
  were saved under.

## Anti-goals

- Anyone other than the owner being able to read or overwrite the quotes.
- An import silently duplicating, changing or deleting existing quotes.

## Approaches considered

- **A. Automatic import from DMs on startup.** Hands-off, but more moving
  parts.
- **B. `/export` + `/import` commands. Chosen by the owner.** Explicit, works
  even if the DM history is gone, and costs one manual step.
- **C. A plus scheduled backups.** The most to build.

## Design

- **On/off switch:** a new `ENABLE_TRANSFER_COMMANDS` env var, false by
  default. When it is off, `/export` and `/import` are left out of the command
  list the bot registers at startup (`SetGuildCommands` replaces the whole
  list), so they disappear from Discord entirely. The bot also ignores them if
  they are called anyway. Turning it on and restarting brings them back.
- **Owner check:** a new `OWNER_ID` env var holds the owner's Discord user ID.
  Both commands refuse (ephemerally) for anyone else, and refuse for everyone
  if `OWNER_ID` is not set. Both skip channel restrictions and cooldowns.
- **Export file:** JSON, named `quotebot-export-YYYY-MM-DD.json`:
  `{"version": 1, "exported_at": "<RFC 3339>", "quotes": [{"id", "guild_id", "quote"}]}`.
  It holds every quote in the database, not just the current server's.
- **`/export`:** defers the reply, reads all quotes, and DMs the file to the
  owner. Replies ephemerally with "Sent N quotes to your DMs". If the DM
  fails (for example DMs from server members are off), it attaches the file
  to the ephemeral reply instead and says so.
- **`/import file:<attachment>`:** defers the reply, downloads the attachment
  (refusing anything over 25 MB or not in the format above), and inserts
  every quote in one transaction with `ON CONFLICT (id) DO NOTHING`. Replies
  with "Imported N new quotes, skipped M that already existed". Running it
  twice is harmless. Any invalid entry aborts the whole import with nothing
  written.
- **Migration order:**
  1. Deploy the new image to the old instance via Portainer, with `OWNER_ID`
     and `ENABLE_TRANSFER_COMMANDS=true` set.
  2. Run `/export`.
  3. Stop the old container.
  4. Start the new instance with the same two settings.
  5. Run `/import` with the file.
  6. Turn `ENABLE_TRANSFER_COMMANDS` off on the new instance and restart, so
     the commands disappear.

  Both instances must never run at the same time with the same token.

## Calls made by Claude (open to change)

- Both commands are hidden in Discord from members without Administrator
  permission, on top of the `OWNER_ID` check. If the owner is not an admin
  in the server, this would hide the commands from them too.
- Export covers all servers' quotes, since the file is a full backup.
- Discord limits bot uploads to about 10 MB, which is roughly 50,000 quotes.
  That's far more than expected, so no splitting is planned.

## Left to the builder

- Package layout, error messages, and tests: the format round-trip, owner
  checks, and the database insert using the existing CI Postgres.
- README section with the migration steps.

# QuoteBot

QuoteBot is a Discord bot that allows users to add and retrieve quotes. It is built using Go and the Disgo library.

## Features
- Add quotes via a modal interaction
- Retrieve and display quotes

## Prerequisites
- Docker
- A Discord bot token

## Getting Started

### Clone the Repository

```sh
git clone https://github.com/yourusername/quotebot-go.git
cd quotebot-go
```

## Running with Docker Compose / Portainer

`docker-compose.yml` runs the bot together with its own Postgres database. The
database is set up and connected automatically and isn't exposed outside the
stack, so you only need to provide the Discord settings:

| Variable | Required | Description |
|---|---|---|
| `TOKEN` | yes | Discord bot token |
| `GUILD_ID` | yes | ID of the server the bot runs in |
| `OWNER_ID` | for `/export` and `/import` | Your Discord user ID |
| `NEW_QUOTE_WEBHOOK` | no | Webhook URL that new quotes are posted to |
| `ALLOWED_CHANNELS` | no | `command:channelID` pairs, comma separated |
| `COOLDOWN_DURATION` | no | Per-user command cooldown, default `60s` |
| `ENABLE_TRANSFER_COMMANDS` | no | `true` registers `/export` and `/import`, default `false` |
| `PG_DB_PASS` | no | Database password inside the stack, default `quotebot` |

- **Docker Compose:** copy `.env.sample` to `.env`, fill it in, and run `docker compose up -d`.
- **Portainer:** create a stack from `docker-compose.yml` and set the variables above in the stack's environment variables.

To find your user ID, turn on Developer Mode in Discord (Settings → Advanced),
then right-click your name and choose **Copy User ID**.

## Moving the bot to a new host

`/export` and `/import` copy every quote from one instance to another. They
only exist while `ENABLE_TRANSFER_COMMANDS=true`, and only the user in
`OWNER_ID` can run them.

1. **Publish a GitHub release first.** That builds the `:latest` image
   that contains these commands.
2. **On the old instance**, set `OWNER_ID` and `ENABLE_TRANSFER_COMMANDS=true`
   and redeploy with the latest image. In Portainer, *Recreate* keeps the old
   settings, so use one of these instead:
   - **Deployed as a stack:** open the stack, add the two variables under
     *Environment variables*, then *Update the stack* with *Re-pull image*
     turned on.
   - **Deployed as a single container:** open the container, choose
     *Duplicate/Edit*, add the two variables under *Advanced container
     settings → Env*, turn on *Always pull the image*, then *Deploy the
     container* and confirm replacing the old one.
3. Run `/export`. The bot DMs you `quotebot-export-YYYY-MM-DD.json`. Save it.
4. **Stop the old container.** Never run two instances with the same token
   at once: both would answer every command.
5. **Start the new instance** (for example the compose stack above) with the
   same `OWNER_ID` and `ENABLE_TRANSFER_COMMANDS=true`. Run `/import` and
   attach the file. Quotes that already exist are skipped, so running it
   twice is harmless.
6. **Turn `ENABLE_TRANSFER_COMMANDS` off** (or remove it) and restart. The
   commands disappear from Discord.

You can turn the switch on at any time and run `/export` to take a backup.

## TODO List

- [x] **Add Quote Command**: Implement the command to add quotes via a modal interaction.
  - [x] **Display UUID**: add ability to show id for the quote.
- [x] **Database Integration**: Set up and integrate a database to store quotes.
- [x] **Error Handling**: Improve error handling and logging throughout the application.
- [ ] **Retrieve Quote Command**: Implement the command to retrieve and display quotes based off of id.
- [ ] **Multi Quote Command**: Implement the command to retrieve and display multiple quotes.
- [ ] **Unit Tests**: Write unit tests for the commands and database interactions.
- [ ] **Docker Compose**: Create a Docker Compose file to simplify the setup of the development environment.

Feel free to contribute by opening an issue or submitting a pull request!
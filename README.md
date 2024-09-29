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

## TODO List

- [x] **Add Quote Command**: Implement the command to add quotes via a modal interaction.
  - [ ] **Display UUID**: add ability to show id for the quote.
- [x] **Database Integration**: Set up and integrate a database to store quotes.
- [x] **Error Handling**: Improve error handling and logging throughout the application.
- [ ] **Retrieve Quote Command**: Implement the command to retrieve and display quotes based off of id.
- [ ] **Multi Quote Command**: Implement the command to retrieve and display multiple quotes.
- [ ] **Unit Tests**: Write unit tests for the commands and database interactions.
- [ ] **Docker Compose**: Create a Docker Compose file to simplify the setup of the development environment.

Feel free to contribute by opening an issue or submitting a pull request!
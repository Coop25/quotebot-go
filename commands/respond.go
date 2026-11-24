package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

// noMentions stops user-submitted quote text from pinging users, roles or
// @everyone when the bot repeats it.
var noMentions = &discord.AllowedMentions{
	Parse: []discord.AllowedMentionType{},
	Roles: []snowflake.ID{},
	Users: []snowflake.ID{},
}

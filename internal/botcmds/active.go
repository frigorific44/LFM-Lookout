package botcmds

import (
	"lfm_lookout/internal/botenv"

	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var ActiveHelp = discordgo.MessageEmbed{
	Title: "Active Command",
	Description: "*[prefix]active*\n\n" +
		"Returns the user's active queries.\n" +
		"Ex: `lo!active`",
}

// [prefix]active
// Retrieves the user's active Lookout queries from the query database and
// returns the user the formatted listing in a message.
func Active(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv) {
	queries, err := env.Repo.FindByAuthor(message.Author.ID)
	if err != nil {
		env.Log.Error(
			"Error retrieving user's queries.",
			zap.Error(err))
	}

	if len(queries) > 0 {
		fields := make([]*discordgo.MessageEmbedField, len(queries))
		k := 0
		for i := range queries {
			fields[k] = &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("ID: %X", queries[i].ID),
				Value: fmt.Sprintf("%s\n*Duration:* %s", queries[i].Query, queries[i].TTL.String()),
			}
			k++
		}
		embed := &discordgo.MessageEmbed{
			Title:  "Queries",
			Fields: fields,
		}
		session.ChannelMessageSendEmbed(message.ChannelID, embed)
	} else {
		session.ChannelMessageSend(message.ChannelID, "No active queries found.")
	}
}

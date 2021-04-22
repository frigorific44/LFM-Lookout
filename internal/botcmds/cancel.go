package botcmds

import (
  "github.com/bwmarrin/discordgo"
  "lfm_lookout/internal/botenv"
)

// [prefix]cancel [query id]
// Removes the specified Lookout query for the query database if it exists and
// belongs to the user.
func Cancel(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  session.ChannelMessageSend(message.ChannelID, "cancel")
}

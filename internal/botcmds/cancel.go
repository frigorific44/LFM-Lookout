package botcmds

import (
  "github.com/bwmarrin/discordgo"
  "lfm_lookout/internal/botenv"
)

func Cancel(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  session.ChannelMessageSend(message.ChannelID, "cancel")
}

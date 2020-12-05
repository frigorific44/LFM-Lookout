package botcmds

import (
  "lfm_lookout/internal/botenv"
  // "lfm_lookout/internal/lodb"
  // "log"
  // "time"

  "github.com/bwmarrin/discordgo"
)

func Lookout(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  // var q lodb.LoQuery
  // q = lodb.LoQuery{
  //   AuthorID: message.Author.ID,
  //   ChannelID: message.ChannelID,
  //   Onetime: false,
  //   Timestamp: time.Now(),
  //   Query: message.Content
  // }
  // err := env.Repo.Save(q)
  // if err != nil {
  //   env.Log.Error(err)
  //   session.ChannelMessageSend(message.ChannelID, "Oh dear, it seems like there was a problem.")
  // }
  // env.Log.Info("New query.")
  // session.ChannelMessageSend(message.ChannelID, "Lookout query saved.")
  session.ChannelMessageSend(message.ChannelID, "lookout")
}

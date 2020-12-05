package botcmds

import (
  "github.com/bwmarrin/discordgo"
  "lfm_lookout/internal/botenv"
  // "lfm_lookout/internal/lodb"
  // "strings"
)

func Active(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv) {
  // if message.Author.ID {
  //   queries, err := env.Repo.FindByAuthor(message.Author.ID)
  //   if err != nil {
  //     env.Log.Error(err)
  //   }
  //
  //   if len(queries) > 0 {
  //     queriesText := []string{}
  //     for i := range queries {
  //       text := queries[i].String()
  //       queriesText = append(queriesText, text)
  //     }
  //     resultText := strings.Join(queriesText, "\n")
  //     session.ChannelMessageSend(message.ChannelID, resultText)
  //   } else {
  //     session.ChannelMessageSend(message.ChannelID, "No results found.")
  //   }
  // } else {
  //   env.Log.Warn("No Author ID found.")
  //   session.ChannelMessageSend(message.ChannelID, "You don't seem to have an ID. Are you sure you're real?")
  // }
  session.ChannelMessageSend(message.ChannelID, "active")
}

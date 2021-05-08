package botcmds

import (
  "lfm_lookout/internal/botenv"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
)

// [prefix]active
// Retrieves the user's active Lookout queries from the query database and
// returns the user the formatted listing in a message.
func Active(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv) {
  fmt.Println("Active command received.")
  defer fmt.Println("Active command processed.")
  queries, err := env.Repo.FindByAuthor(message.Author.ID)
  if err != nil {
    env.Log.Error(err)
  }

  if len(queries) > 0 {
    queriesText := []string{}
    for i := range queries {
      text := queries[i].String()
      queriesText = append(queriesText, text)
    }
    resultText := strings.Join(queriesText, "\n")
    session.ChannelMessageSend(message.ChannelID, resultText)
  } else {
    session.ChannelMessageSend(message.ChannelID, "No active queries found.")
  }
}

package botcmds

import (
  "lfm_lookout/internal/lodb"

  "fmt"
  "strings"
  "unicode/utf8"

  "github.com/bwmarrin/discordgo"
  "lfm_lookout/internal/botenv"
)

// [prefix]cancel [query id]
// Removes the specified Lookout query for the query database if it exists and
// belongs to the user.
func Cancel(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  // Parse out the index rune.
  r, _ := utf8.DecodeLastRuneInString(strings.TrimSpace(message.Content))
  // Check the rune.
  if (r < lodb.IDMIN || r > lodb.IDMAX) {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("The ID %s is not within an acceptable range.", string(r)))
    return
  }
  // Delete.
  err := env.Repo.Delete(message.Author.ID, r)
  if err != nil {
    env.Log.Error(err)
    session.ChannelMessageSend(message.ChannelID, "The was a problem trying to delete that query.")
    return
  }
  session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Query %s was canceled.", string(r)))
}

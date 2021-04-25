package botcmds

import (
  "lfm_lookout/internal/lodb"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
  "lfm_lookout/internal/botenv"
)

// [prefix]cancel [query id]
// Removes the specified Lookout query for the query database if it exists and
// belongs to the user.
func Cancel(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  // Parse out the index byte.
  trimmed := []byte(strings.TrimSpace(message.Content))
  b := trimmed[len(trimmed)-1]
  // Check the byte
  if (b < lodb.IDMIN || b > lodb.IDMAX) {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("The ID is not within an acceptable range."))
    return
  }
  // Send the delete request.
  env.DelChan <- botenv.DeleteRequest{AuthorID:message.Author.ID,Index:b,}
}

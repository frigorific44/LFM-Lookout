package botcmds

import (
  "lfm_lookout/internal/botenv"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
)

var GroupsHelp string = "groups [server]"
// [prefix]groups [server]
// Retrieves the server entry in audit for the specified server, if the entry
// exists. Formats entry and then sends it to the requesting user.
func Groups(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  fmt.Println("Groups command received.")
  defer fmt.Println("Groups command processed.")
  strTokens := strings.Fields(message.Content[len(env.Config.Prefix):])
  if len(strTokens) <= 1 {
    session.ChannelMessageSend(message.ChannelID, "No server argument found.")
    return
  }
  server := strings.Title(strings.ToLower(strings.TrimSpace(strTokens[1])))
  // Search for a matching server.
  env.AuditLock.RLock()
  defer env.AuditLock.RUnlock()
  serverMatch, exists := env.Audit.Map[server]
  if !exists {
    session.ChannelMessageSend(message.ChannelID, "A server with that name was not found.")
    return
  }
  // With server index found, construct a formatted strings of the groups.
  b := strings.Builder{}
  for _, sGroup := range serverMatch {
    b.WriteString(sGroup.Group.String())
    b.WriteString("\n\n")
  }
  embed := discordgo.MessageEmbed{Title: server, Description: b.String()}
  session.ChannelMessageSendEmbed(message.ChannelID, &embed)
}

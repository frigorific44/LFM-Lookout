package botcmds

import (
  "lfm_lookout/internal/botenv"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
)

func Groups(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  strTokens := strings.Fields(message.Content[len(env.Config.Prefix):])
  if len(strTokens) <= 1 {
    session.ChannelMessageSend(message.ChannelID, "No server argument found.")
    return
  }
  server := strings.Title(strings.ToLower(strings.TrimSpace(strTokens[1])))
  // Search for a matching server.
  env.AuditLock.RLock()
  index := -1
  for i := range env.Audit.Servers {
    if server == env.Audit.Servers[i].Name {
      index = i
    }
  }
  if index == -1 {
    session.ChannelMessageSend(message.ChannelID, "A server with that name was not found.")
    return
  }
  // With server index found, construct a formatted strings of the groups.
  groups := ""
  for _, g := range env.Audit.Servers[index].Groups {
    groups = groups + fmt.Sprintf("```ini\n%s```", g.String())
  }
  env.AuditLock.RUnlock()
  session.ChannelMessageSend(message.ChannelID, groups)
}

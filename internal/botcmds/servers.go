package botcmds

import (
  "lfm_lookout/internal/botenv"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
)

var ServersHelp string = "servers\nReturns the names of all currently active servers."
// [prefix]servers
// Retrieves all server names currently contained as keys in the audit map.
func Servers(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  env.AuditLock.RLock()
  defer env.AuditLock.RUnlock()
  var b strings.Builder

  i := 0
  b.WriteString("```\n")
  for k := range env.Audit.Map {
    fmt.Fprintf(&b, "%s\n", k)
    i++
  }
  b.WriteString("```")
  session.ChannelMessageSend(message.ChannelID, b.String())
}

package botcmds

import (
  "lfm_lookout/internal/botenv"

  "fmt"
  "strings"

  "github.com/bwmarrin/discordgo"
)

var ServersHelp = discordgo.MessageEmbed{
  Title: "Servers Command",
  Description:
  "*[prefix]servers*\n\n" +
  "Returns the list of servers.\n" +
  "Ex: `lo!servers`",
}
// [prefix]servers
// Retrieves all server names currently contained as keys in the audit map.
func Servers(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  env.AuditLock.RLock()
  defer env.AuditLock.RUnlock()
  var b strings.Builder

  i := 0
  for k := range env.Audit.Map {
    fmt.Fprintf(&b, "%s\n", k)
    i++
  }
  embed := discordgo.MessageEmbed{Title: "Servers", Description: b.String(),}
  session.ChannelMessageSendEmbed(message.ChannelID, &embed)
}

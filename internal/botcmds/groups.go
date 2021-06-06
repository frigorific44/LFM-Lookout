package botcmds

import (
	"lfm_lookout/internal/botenv"

	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var GroupsHelp = discordgo.MessageEmbed{
	Title: "Groups Command",
	Description: "*[prefix]groups [server]*\n\n" +
		"Returns a list of current groups in the specified server.\n" +
		"Ex: `lo!groups Cannith`",
}

// [prefix]groups [server]
// Retrieves the server entry in audit for the specified server, if the entry
// exists. Formats entry and then sends it to the requesting user.
func Groups(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv) {
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
	// Sort group keys by max level.
	keys := make([]string, len(serverMatch))
	i := 0
	for k := range serverMatch {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return serverMatch[keys[i]].Group.MinLevel > serverMatch[keys[j]].Group.MinLevel
	})
	// With server index found, construct a formatted strings of the groups.
	var b strings.Builder
	for i := range keys {
		b.WriteString(serverMatch[keys[i]].Group.String())
		b.WriteString("\n\n")
	}
	embed := discordgo.MessageEmbed{Title: server, Description: b.String()}
	session.ChannelMessageSendEmbed(message.ChannelID, &embed)
}

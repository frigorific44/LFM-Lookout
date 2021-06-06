package botcmds

import (
	"lfm_lookout/internal/botenv"
	"lfm_lookout/internal/lodb"

	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var CancelHelp = discordgo.MessageEmbed{
	Title: "Cancel Command",
	Description: "*[prefix]cancel [query id]*\n\n" +
		"Cancels a user's query of the specified ID.\n" +
		"Ex: `lo!cancel 00F`",
}

// [prefix]cancel [query id]
// Removes the specified Lookout query for the query database if it exists and
// belongs to the user.
func Cancel(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv) {
	// Parse out the index rune.
	f := (strings.Fields(message.Content))
	id := f[len(f)-1]
	i, errConv := strconv.ParseInt(id, 16, 32)
	if errConv != nil {
		return
	}
	r := rune(i)
	// Check the rune.
	if r < lodb.IDMIN || r > lodb.IDMAX*lodb.TICKPERIOD {
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("The ID %s is not within an acceptable range.", string(r)))
		return
	}
	// Delete.
	err := env.Repo.Delete(message.Author.ID, r)
	if err != nil {
		env.Log.Error(
			"Error deleting user's query.",
			zap.Error(err))
		session.ChannelMessageSend(message.ChannelID, "The was a problem trying to delete that query.")
		return
	}
	session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Query %X was canceled.", r))
}

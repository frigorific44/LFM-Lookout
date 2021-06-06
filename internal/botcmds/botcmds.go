package botcmds

import (
	"lfm_lookout/internal/botenv"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Cmd     func(*discordgo.Session, *discordgo.MessageCreate, *botenv.BotEnv)
	HelpMsg discordgo.MessageEmbed
}

var Commands = map[string]Command{
	"active":  Command{Cmd: Active, HelpMsg: ActiveHelp},
	"cancel":  Command{Cmd: Cancel, HelpMsg: CancelHelp},
	"groups":  Command{Cmd: Groups, HelpMsg: GroupsHelp},
	"lookout": Command{Cmd: Lookout, HelpMsg: LookoutHelp},
	"servers": Command{Cmd: Servers, HelpMsg: ServersHelp},
}

var CommandsMsg = discordgo.MessageEmbed{
	Title: "Commands Help",
	Description: "active\ncancel\ngroups\nlookout\nservers\n\n" +
		"For more information on a command, use `lo!help [command]`\n" +
		"Ex: `lo!help groups`",
}

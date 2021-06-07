# Lookout!

***Lookout!*** is a Discord Bot for the DDO community, allowing players to view current group advertisements and set timed search queries. Anytime a group is found matching a user's query, they will be notified on Discord with the group's information, helping players find the groups they need, and facilitating interconnectedness in the community—without requiring that you stayed logged in.

To use *Lookout!*, feel free to [invite](https://discord.com/oauth2/authorize?client_id=771959114338926633&scope=bot&permissions=0) it to a server of your own, where you can start exploring its functionality with `lo!help`.

## Setup

To run a *Lookout!* bot of your own, make sure you have Go 1.16.5 or higher, or at least a version which supports the module functionality. You will also need a Discord account, and have a bot application registered. 

You can create a bot application in the [Developer Portal](https://discord.com/developers/applications), where you can create a new application. With an application created, go to its settings and select bot, and confirm that you wish to add a bot presence to your application. In here, you will then want to copy your bot token, or at least note its presence for later.

With this repository's files downloaded, the only thing that you'll need to change is the bot token inside of `config.json`. Remove the place-holder and insert your own bot's token.

With a terminal open on the directory you downloaded the repository to, you can either `go run main.go`, or `go build main.go` and run the generated executable.

In order to see your bot running, it will have to be invited to at least one server you are present in. From there, you can enter commands from one of the server's channels, or direct message your bot and issue commands from there.




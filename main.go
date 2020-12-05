package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"reflect"

	dg "github.com/bwmarrin/discordgo"
	logrus "github.com/sirupsen/logrus"

	"lfm_lookout/internal/audit"
	"lfm_lookout/internal/botcmds"
	"lfm_lookout/internal/botenv"
)


func main() {
	// Open file for logging.
	currTime := time.Now()
	logPath := "logs/"+currTime.Format("2006-01-02")+".log"
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Fatal(err)
	}
	defer file.Close()
	// Create a logging object which can be passed around (safely).
	var log = &logrus.Logger{
		Out: file,
		Formatter: new(logrus.JSONFormatter),
		Level: logrus.DebugLevel,
	}
	log.Info("Logging to file.")
	// Prepare the database.
	// db, err := sql.Open("sqlite3",
	// 	"./lookout.db")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()
	// Check that the database is available and accessible.
	// err = db.Ping()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// Instantiate BotEnv containing information for commands.
	// repo := lodb.NewLoRepo(&log, &db)
	// defer repo.Close()
	botEnv := botenv.BotEnv{Log: log}
	// Load the config.json file.
	io, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	// Load JSON into botenv:config.
	json.Unmarshal(io, &botEnv.Config)
	// Get current groups from playeraudit.com
	audit, err := audit.Groups()
	if err != nil {
		log.Fatal(err)
	}
	botEnv.Audit = audit
	// Create a new Discord session using the provided bot token.
	bot, err := dg.New("Bot " + botEnv.Config.Token)
	if err != nil {
		log.Fatal(err)
	}
	// Embed BotEnv in LookoutEnv so BotEnv can be passed to commands.
	loEnv := LookoutEnv{Env: &botEnv}
	// Register the messageCreate func as a callback for MessageCreate events.
	bot.AddHandler(loEnv.messageCreate)
	// We only care about message events, so let's make that clear.
	bot.Identify.Intents = dg.MakeIntent(dg.IntentsGuildMessages | dg.IntentsDirectMessages)
	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		log.Fatal(err)
	}
	log.Debug(botcmds.Functions)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	// Cleanly close down the Discord session.
	bot.Close()
}

type LookoutEnv struct {
	Env *botenv.BotEnv
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (env *LookoutEnv) messageCreate(s *dg.Session, m *dg.MessageCreate) {
	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore messages from other bots.
	if m.Author.Bot {
		return
	}
	// Ignore messages from webhooks.
	if m.WebhookID != "" {
		return
	}
	// If it's a message we care about, check if it's a command, and execute.
	if strings.HasPrefix(m.Content, env.Env.Config.Prefix) {
		var strTokens = strings.Fields(m.Content[len(env.Env.Config.Prefix):])
		if len(strTokens) > 0 {
			var c = strings.Title(strings.ToLower(strTokens[0]))
			// Reflection is used so commands need not be manually hard-coded.
			// Instead, a precompiler is used to generate a map to command functions.
			reflection, ok := botcmds.Functions[c]
			if ok {
				in := make([]reflect.Value, 3)
      	in[0] = reflect.ValueOf(s)
				in[1] = reflect.ValueOf(m)
				in[2] = reflect.ValueOf(env.Env)
				reflection.Call(in)
			}
		}
	}
}

package main

import (
	"lfm_lookout/internal/audit"
	"lfm_lookout/internal/botcmds"
	"lfm_lookout/internal/botenv"
	"lfm_lookout/internal/lodb"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"reflect"

	"github.com/blevesearch/bleve"
	dg "github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)


type SearchableGroup struct {
	Server string
	Group audit.Group
}

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
	auditLock := new(sync.RWMutex)
	botEnv := botenv.BotEnv{Log: log, AuditLock: auditLock}
	// Load the config.json file.
	io, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	// Load JSON into botenv:config.
	json.Unmarshal(io, &botEnv.Config)
	// Initialize LoRepo
	repo, err := lodb.NewLoRepo("/tmp/badger")
	if err != nil {
		log.Panic(err)
	}
	// TODO: Clean up orphan query and return entries.
	// Get current groups from playeraudit.com
	currAudit, err := audit.Groups()
	if err != nil {
		log.Fatal(err)
	}
	botEnv.Audit = currAudit
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
	// Periodically update botEnv.Audit.
	auditTicker := time.NewTicker(time.Second * 31)
	quit := make(chan bool)
	go func() {
		for {
			select {
			// Update audit, cull expired queries, then run queries on audit
			case <- auditTicker.C:
				// Update audit.
				newAudit, err := audit.Groups()
				if err != nil {
					botEnv.Log.Error(err)
				}
				botEnv.AuditLock.Lock()
				botEnv.Audit = newAudit
				botEnv.AuditLock.Unlock()
				botEnv.Log.Info("Audit updated.")
				// Open a new index.
				mapping := bleve.NewIndexMapping()
				index, err := bleve.NewMemOnly(mapping)
				defer index.Close()
				if err != nil {
					botEnv.Log.Error(err)
					continue
				}
				batch := index.NewBatch()
				botEnv.AuditLock.RLock()
				for _, server := range botEnv.Audit.Servers {
					for _, group := range server.Groups {
						batch.Index(group.Id, SearchableGroup{
							Server: server.Name,
							Group: group,
						})
					}
				}
				botEnv.AuditLock.RUnlock()
				if err := index.Batch(batch); err != nil {
					botEnv.Log.Error(err)
					continue
				}
				// Run queries on current groups.
				errReIt := repo.View(func(txn *badger.Txn) error {
  				it := txn.NewIterator(badger.DefaultIteratorOptions)
					defer it.Close()
					prefix := []byte("query")
					for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
						item := it.Item()
						k := item.Key()
						err := item.Value(func(v []byte) error {
							// If query Server field does not match any servers, alert User
							// and mark for deletion.
							query := bleve.NewQueryStringQuery(string(v))
							search := bleve.NewSearchRequest(query)
							searchResults, err := index.Search(search)
							if err != nil {
								botEnv.Log.Error(err)
								continue
							}
							for _, match := range searchResults.Hits {
								doc, err := index.Document(match.ID)
								if err != nil {
									botEnv.Log.Error(err)
									continue
								}
								// Assuming the low frequency in new matches, and the low
								// frequency in which the Groups command is invoked, tying up
								// the Audit structure should pose no hinderance.
								// TODO: Investigate this.
							}
						})
						if err != nil {
							return err
						}
					}
  				return nil
				})
				if errReIt != nil {
					botEnv.Log.Error(errReIt)
				}
			case <- incoming:
				// TODO: Check that Server matches an existing server.
				continue
			case <- delete:
				continue
			case <- quit:
				auditTicker.Stop()
				return
			}
		}
	} ()
	defer close(quit)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	// Cleanly close down the Discord session.
	quit <- true
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

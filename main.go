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
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/blevesearch/bleve"
	dg "github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
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
	botEnv.Repo = repo
	// TODO: Clean up orphan query and return entries.
	// Get current groups from playeraudit.com
	currAudit, err := audit.Groups()
	if err != nil {
		log.Fatal(err)
	} else {
		botEnv.Audit = AuditToMap(currAudit)
	}
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
	incoming := make(chan lodb.LoQuery, 25)
	botEnv.LoChan = incoming
	go func() {
		for {
			select {
			// Update audit, cull expired queries, then run queries on audit
			case <- auditTicker.C:
				// Update audit.
				newAudit, err := audit.Groups()
				if err != nil {
					botEnv.Log.Error(err)
				} else{
					botEnv.AuditLock.Lock()
					botEnv.Audit = AuditToMap(newAudit)
					botEnv.AuditLock.Unlock()
					botEnv.Log.Info("Audit updated.")
				}
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
				for _, serverMap := range botEnv.Audit {
					for idStr, sGroup := range serverMap {
						batch.Index(idStr, sGroup)
					}
				}
				botEnv.AuditLock.RUnlock()
				if err := index.Batch(batch); err != nil {
					botEnv.Log.Error(err)
					continue
				}
				// Run queries on current groups.
				errReIt := repo.GetView(func(txn *badger.Txn) error {
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
								return err
							}
							botEnv.AuditLock.RLock()
							defer botEnv.AuditLock.RUnlock()
							// Review each match and act accordingly.
							for _, match := range searchResults.Hits {
								// TODO: Check that matched group hasn't already matched to this query.
								found := false
								// Iterate through servers to find the corresponding group.
								for _, server := range botEnv.Audit {
									sGroup, exists := server[match.ID]
									if (exists) {
										found = true
										chanKey := strings.Replace(string(k), "query", "return", 1)
										chanItem, err := txn.Get([]byte(chanKey))
										if (err != nil) {
											botEnv.Log.Error(err)
											break
										}
									  var channel []byte
									  errVal := chanItem.Value(func(val []byte) error {
									    channel = append([]byte{}, val...)
									    return nil
									  })
										if errVal != nil {
											botEnv.Log.Error(errVal)
											break
										}
										m := fmt.Sprintf("You got a hit!\n```ini\n%s```", sGroup.Group.String())
										bot.ChannelMessageSend(string(channel), m)
										break
									}
								}
								if !found {
									botEnv.Log.Error("Group match was not found in Audit map.")
								}
							}
							return nil
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
			case q := <- incoming:
				// Check that Server matches an existing server.
				re := regexp.MustCompile(`Server:\s*(\w+)`)
				sMatch := re.FindStringSubmatch(q.Query)
				if (sMatch == nil) {continue}
				s := sMatch[1]
				_, exists := botEnv.Audit[s]
				if !exists {
					bot.ChannelMessageSend(q.ChannelID, "The requested query does not seem to specify a server which actually exists.")
					continue
				}
				// Save query to the repository.
				err := repo.Save(q)
				if err != nil {
					botEnv.Log.Error(err)
					bot.ChannelMessageSend(q.ChannelID, "Oh dear, it seems like there was a problem.")
				} else {
					botEnv.Log.Info("New query.")
					bot.ChannelMessageSend(q.ChannelID, "Lookout query saved.")
				}
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

func AuditToMap(audit *audit.Audit) map[string]map[string]botenv.SearchableGroup {
	var newMap = make(map[string]map[string]botenv.SearchableGroup)
	for _, server := range audit.Servers {
		newMap[server.Name] = make(map[string]botenv.SearchableGroup)
		for _, group := range server.Groups {
			newMap[server.Name][fmt.Sprintf("%d", group.Id)] = botenv.SearchableGroup{
				Server: server.Name,
				Group: group,
			}
		}
	}
	return newMap
}

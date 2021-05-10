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
	tickLock := new(sync.RWMutex)
	n := time.Now()
	tick := rune((n.Hour() * 120) + (n.Minute() * 2) + (n.Second() / 30))
	botEnv := botenv.BotEnv{Log: log, AuditLock: auditLock, Tick: tick, TickLock: tickLock}
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
	cmdsString := "active\ncancel\ngroups\nlookout\nservers"
	loEnv := LookoutEnv{Env: &botEnv, CmdsString: cmdsString}
	// Register the messageCreate func as a callback for MessageCreate events.
	bot.AddHandler(loEnv.messageCreate)
	// We only care about message events, so let's make that clear.
	bot.Identify.Intents = dg.MakeIntent(dg.IntentsGuildMessages | dg.IntentsDirectMessages)
	// Open a websocket connection to Discord and begin listening.
	err = bot.Open()
	if err != nil {
		log.Fatal(err)
	}
	log.Debug(botcmds.Commands)
	// Periodically update botEnv.Audit.
	auditTicker := time.NewTicker(time.Second * 31)
	quit := make(chan bool)
	go func() {
		for {
			select {
			// Update audit, cull expired queries, then run queries on audit
			case <- auditTicker.C:
				// Update audit.
				startTotal := time.Now()
				newAudit, err := audit.Groups()
				if err != nil {
					botEnv.Log.Error(err)
				} else{
					botEnv.AuditLock.Lock()
					prevAudit := botEnv.Audit
					botEnv.Audit = AuditToUpdatedMap(newAudit, prevAudit)
					botEnv.AuditLock.Unlock()
					botEnv.Log.Info("Audit updated.")
				}
				// Open a new index.
				startIndex := time.Now()
				mapping := bleve.NewIndexMapping()
				index, err := bleve.NewMemOnly(mapping)
				defer index.Close()
				if err != nil {
					botEnv.Log.Error(err)
					continue
				}
				batch := index.NewBatch()
				botEnv.AuditLock.RLock()
				for _, serverMap := range botEnv.Audit.Map {
					for idStr, sGroup := range serverMap {
						batch.Index(idStr, sGroup)
					}
				}
				botEnv.AuditLock.RUnlock()
				if err := index.Batch(batch); err != nil {
					botEnv.Log.Error(err)
					continue
				}
				startSearch := time.Now()
				botEnv.TickLock.Lock()
				currTick := botEnv.Tick
				botEnv.Tick = lodb.NextTickRune(currTick)
				botEnv.TickLock.Unlock()
				// Run queries on current groups.
				errReIt := repo.GetView(func(txn *badger.Txn) error {
  				it := txn.NewIterator(badger.DefaultIteratorOptions)
					defer it.Close()
					prefix := []byte("query")
					for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
						item := it.Item()
						k := item.Key()
						_, t := lodb.DecodeFinalRune(lodb.GetIDFromKey(string(k)))
						// If the query is from the next tick, defer until then.
						botEnv.TickLock.RLock()
						if t == botEnv.Tick {continue}
						botEnv.TickLock.RUnlock()
						err := item.Value(func(v []byte) error {
							queryBase := bleve.NewQueryStringQuery(string(v))
							// If query is from the preceding tick, run against all
							// groups, otherwise, run against only fresh ones.
							var search *bleve.SearchRequest
							if t != currTick {
								onlyFresh := bleve.NewBoolFieldQuery(true)
								onlyFresh.SetField("Fresh")
								query := bleve.NewConjunctionQuery(queryBase, onlyFresh)
								search = bleve.NewSearchRequest(query)
							} else {
								search = bleve.NewSearchRequest(queryBase)
							}
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
								for _, server := range botEnv.Audit.Map {
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
										r := lodb.GetIDFromKey(string(k))
										m := fmt.Sprintf("**ID: ** %X\n```%s```", r, sGroup.Group.String())
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
				stop := time.Now()
				botEnv.Log.WithFields(logrus.Fields{
					"total_dur": stop.Sub(startTotal).String(),
					"audit_dur": startIndex.Sub(startTotal).String(),
					"index_dur": startSearch.Sub(startIndex).String(),
					"search_dur": stop.Sub(startSearch).String(),
				}).Info("Audit ticker loop.")
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
	CmdsString string
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
			var c = strings.ToLower(strTokens[0])
			if c == "help" {
				switch len(strTokens) {
				case 1:
					s.ChannelMessageSend(m.ChannelID, env.CmdsString)
				case 2:
					command, ok := botcmds.Commands[strings.ToLower(strTokens[1])]
					if ok {s.ChannelMessageSend(m.ChannelID, command.HelpMsg)}
				default:
					return
				}
			}
			command, ok := botcmds.Commands[c]
			if ok {
				command.Cmd(s, m, env.Env)
			}
		}
	}
}

func AuditToMap(audit *audit.Audit) botenv.AuditMap {
	var newMap = make(map[string]map[string]botenv.SearchableGroup)
	for _, server := range audit.Servers {
		newMap[server.Name] = make(map[string]botenv.SearchableGroup)
		for _, group := range server.Groups {
			newMap[server.Name][fmt.Sprintf("%d", group.Id)] = botenv.SearchableGroup{
				Server: server.Name,
				Group: group,
				Fresh: true,
			}
		}
	}
	return botenv.AuditMap{Map: newMap,}
}

func AuditToUpdatedMap(audit *audit.Audit, prevMap botenv.AuditMap) botenv.AuditMap {
	var newMap = make(map[string]map[string]botenv.SearchableGroup)
	for _, server := range audit.Servers {
		newMap[server.Name] = make(map[string]botenv.SearchableGroup)
		for _, group := range server.Groups {
			id := fmt.Sprintf("%d", group.Id)
			var f bool
			other, ok := prevMap.Map[server.Name][id]
			if ok {
				f = group.Changed(other.Group)
			} else {
				f = true
			}
			newMap[server.Name][id] = botenv.SearchableGroup{
				Server: server.Name,
				Group: group,
				Fresh: f,
			}
		}
	}
	return botenv.AuditMap{Map: newMap,}
}

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

	"github.com/blevesearch/bleve/v2"
	dg "github.com/bwmarrin/discordgo"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Create a logging object which can be passed around (safely).
	log := getLogger()
	defer log.Sync()
	log.Info("Logging to file.")
	auditLock := new(sync.RWMutex)
	tickLock := new(sync.RWMutex)
	n := time.Now()
	tick := rune((n.Hour() * 120) + (n.Minute() * 2) + (n.Second() / 30))
	botEnv := botenv.BotEnv{Log: log, AuditLock: auditLock, Tick: tick, TickLock: tickLock}
	// Load the config.json file.
	io, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatal(
			"Error loading the config file.",
			zap.Error(err))
	}
	// Load JSON into botenv:config.
	json.Unmarshal(io, &botEnv.Config)
	// Initialize LoRepo
	repo, err := lodb.NewLoRepo("./badger")
	if err != nil {
		log.Panic(
			"Error initializing the query repository.",
			zap.Error(err))
	}
	botEnv.Repo = repo
	// TODO: Clean up orphan query and return entries.
	// Get current groups from playeraudit.com
	currAudit, err := audit.Groups()
	if err != nil {
		log.Fatal(
			"Error getting the groups audit.",
			zap.Error(err))
	} else {
		botEnv.Audit = AuditToMap(currAudit)
	}
	// Create a new Discord session using the provided bot token.
	bot, err := dg.New("Bot " + botEnv.Config.Token)
	if err != nil {
		log.Fatal(
			"Error initializing the bot.",
			zap.Error(err))
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
		log.Fatal(
			"Error connecting the bot.",
			zap.Error(err))
	}
	// Periodically update botEnv.Audit.
	if botEnv.Config.AuditPeriod < 30 {
		log.Panic("Audit period is faster than the PlayerAudit API allows.")
	}
	auditTicker := time.NewTicker(time.Second * time.Duration(botEnv.Config.AuditPeriod))
	quit := make(chan bool)
	go func() {
		for {
			select {
			// Update audit, cull expired queries, then run queries on audit
			case <-auditTicker.C:
				// Update audit.
				startTotal := time.Now()
				newAudit, err := audit.Groups()
				if err != nil {
					botEnv.Log.Error(
						"Error updating the audit.",
						zap.Error(err))
				} else {
					botEnv.AuditLock.Lock()
					prevAudit := botEnv.Audit
					botEnv.Audit = AuditToUpdatedMap(newAudit, prevAudit)
					botEnv.AuditLock.Unlock()
					botEnv.Log.Info("Audit updated.")
				}
				// Open a new index.
				startIndex := time.Now()
				// mapping := bleve.NewIndexMapping()
				mapping, _ := buildIndexMapping()
				index, err := bleve.NewMemOnly(mapping)
				if err != nil {
					botEnv.Log.Error(
						"Error initializing the index.",
						zap.Error(err))
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
					botEnv.Log.Error(
						"Error indexing batch of groups.",
						zap.Error(err))
					continue
				}
				// fmt.Println(index.Fields())
				startSearch := time.Now()
				botEnv.TickLock.Lock()
				currTick := botEnv.Tick
				botEnv.Tick = lodb.NextTickRune(currTick)
				botEnv.TickLock.Unlock()
				// Run queries on current groups.
				var delQ []string
				var qNum int
				errReIt := repo.GetView(func(txn *badger.Txn) error {
					it := txn.NewIterator(badger.DefaultIteratorOptions)
					prefix := []byte("query")
					for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
						qNum++
						item := it.Item()
						key := string(item.Key())
						r := lodb.GetIDFromKey(key)
						_, t := lodb.DecodeFinalRune(r)
						// If the query is from the next tick, defer until then.
						botEnv.TickLock.RLock()
						if t == botEnv.Tick {
							continue
						}
						botEnv.TickLock.RUnlock()
						err := item.Value(func(v []byte) error {
							val := string(v)
							qsStart := time.Now()
							queryBase := bleve.NewQueryStringQuery(val)
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
							search.Fields = []string{"Server"}
							// TODO: Mark queries that take too long to search.
							searchResults, err := index.Search(search)
							qs := time.Since(qsStart)
							if qs > (time.Millisecond * 50) {
								botEnv.Log.Warn(
									"Query took too long to search against.",
									zap.String("query", val),
									zap.Duration("search_t", qs))
								delQ = append(delQ, key)
							}
							botEnv.Log.Debug(
								"Query search.",
								zap.Duration("Search_t", qs))
							if err != nil {
								botEnv.Log.Warn(
									"Query resulted in error upon searching.",
									zap.String("query", val))
								delQ = append(delQ, key)
								return err
							}
							botEnv.AuditLock.RLock()
							mt := time.Now()
							var b strings.Builder // For combining hits into a single message.
							var channel []byte
							if len(searchResults.Hits) > 0 {
								chanKey := strings.Replace(key, "query", "return", 1)
								chanItem, err := txn.Get([]byte(chanKey))
								if err != nil {
									botEnv.Log.Error(
										"Error getting the channel id kv pair.",
										zap.Error(err))
									return err
								}
								errVal := chanItem.Value(func(val []byte) error {
									channel = append([]byte{}, val...)
									return nil
								})
								if errVal != nil {
									botEnv.Log.Error(
										"Error retrieving the channel id value.",
										zap.Error(errVal))
									return errVal
								}
							}
							// Review each match and act accordingly.
							for _, match := range searchResults.Hits {
								sGroup, exists := botEnv.Audit.Map[match.Fields["Server"].(string)][match.ID]
								if exists {
									m := fmt.Sprintf("**ID: %X**, %s\n%s\n", r, sGroup.Server, sGroup.Group.String())
									b.WriteString(m)
								} else {
									botEnv.Log.Warn(
										"Group match was not found in Audit map.",
										zap.String("query", val),
										zap.String("server", match.Fields["Server"].(string)))
								}
							}
							go func(channel string, message string) {
								bot.ChannelMessageSend(channel, message)
							}(string(channel), b.String())
							botEnv.Log.Debug(
								"Match iteration.",
								zap.Duration("matches_t", time.Since(mt)))
							botEnv.AuditLock.RUnlock()
							return nil
						})
						if err != nil {
							botEnv.Log.Error(
								"Error while using query's value.",
								zap.Error(err))
							continue
						}
					}
					it.Close()
					return nil
				})
				if errReIt != nil {
					botEnv.Log.Error(
						"Error while iterating through queries.",
						zap.Error(errReIt))
				}
				stop := time.Now()
				botEnv.Log.Info(
					"Ticker Loop",
					zap.Duration("total", stop.Sub(startTotal)),
					zap.Duration("auditing", startIndex.Sub(startTotal)),
					zap.Duration("indexing", startSearch.Sub(startIndex)),
					zap.Duration("searching", stop.Sub(startSearch)))
				index.Close()
				// Log some of the bot's stats.
				botEnv.Log.Info(
					"Bot Statistics",
					zap.Int("guilds", len(bot.State.Ready.Guilds)),
					zap.Int("queries", qNum))
				// Delete problematic queries.
				for i := range delQ {
					botEnv.Repo.Delete(lodb.GetAuthFromKey(delQ[i]), lodb.GetIDFromKey(delQ[i]))
				}
			case <-quit:
				auditTicker.Stop()
				return
			}
		}
	}()
	defer close(quit)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
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
			var c = strings.ToLower(strTokens[0])
			if c == "help" {
				switch len(strTokens) {
				case 1:
					s.ChannelMessageSendEmbed(m.ChannelID, &botcmds.CommandsMsg)
				case 2:
					command, ok := botcmds.Commands[strings.ToLower(strTokens[1])]
					if ok {
						s.ChannelMessageSendEmbed(m.ChannelID, &command.HelpMsg)
					}
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
				Server:  server.Name,
				Group:   group,
				Members: 1 + len(group.Members),
				Fresh:   true,
			}
		}
	}
	return botenv.AuditMap{Map: newMap}
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
				Server:  server.Name,
				Group:   group,
				Members: 1 + len(group.Members),
				Fresh:   f,
			}
		}
	}
	return botenv.AuditMap{Map: newMap}
}

func getLogger() *zap.Logger {
	core := zapcore.NewCore(
		getEncoder(),
		getWriteSyncer(),
		zap.InfoLevel,
	)
	return zap.New(core)
}

func getWriteSyncer() zapcore.WriteSyncer {
	// lumberjack.Logger is already safe for concurrent use, so we don't need to
	// lock it.
	ws := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./logs/bot.log",
		MaxSize:    500, // megabytes
		MaxBackups: 10,
		MaxAge:     28, // days
	})
	return ws
}

func getEncoder() zapcore.Encoder {
	encConfig := zapcore.EncoderConfig{
		TimeKey:        "t",
		LevelKey:       "lvl",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zapcore.NewJSONEncoder(encConfig)
}

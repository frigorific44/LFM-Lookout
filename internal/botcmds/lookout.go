package botcmds

import (
  "lfm_lookout/internal/botenv"
  "lfm_lookout/internal/lodb"
  "errors"
  "fmt"
  // "log"
  "regexp"
  "strconv"
  "strings"
  "time"

  "github.com/bwmarrin/discordgo"
)


const (
  queryLenMax int = 281
)

var LookoutHelp = discordgo.MessageEmbed{
  Title: "Lookout Command",
  Description:
  "[prefix]lookout Server:[string] Duration:[0h1m-24h0m] (Level:[1-30]) (-/+)term (-/+)\"a phrase\"\n\n" +
  "Saves the query so that for the specified duration, the user will be notified of any matching groups." +
  " Along with terms and phrases searched against all of a group's text, additional fields can be specified—" +
  "similarly to the *Server* and *Duration* fields—with the field name directly followed by a colon.\n" +
  " Valid fields: *Group.Comment, Group.Difficulty, Group.AdventureActive," +
  " Group.Quest.Name, Group.Quest.AdventurePack, Group.Quest.AdventureArea," +
  " Group.Quest.QuestJournalGroup, Group.Quest.GroupSize, Group.Quest.Patron*\n" +
  "Ex: `lo!lookout Server:Cannith Duration:5h Level:30 +Raid +\"Killing Time\"`\n",
}
// [prefix]lookout Server:[string] Duration:[0h1m-24h0m] (level:[1-30]) (-/+)term (-/+)"a phrase"
// A lookout query requires specification of the server and the duration of the query.
// Search terms and phrases will be searched across the whole of a group's
// information. Phrases are delimitted by a surrounding pair of quotes. Terms and
// phrases can optionally be specified as required or exluded, specified by a
// preceding + and -, respectively. Terms and phrases can also be specified by
// the field, of which they will be searched against. These can be specifed by
// the field name, a colon, and then the search term of phrase. Optional search
// fields include Comment, Quest, Difficulty, and Patron.
func Lookout(session *discordgo.Session, message *discordgo.MessageCreate, env *botenv.BotEnv)  {
  fmt.Println("Lookout command received.")
  defer fmt.Println("Lookout command processed.")
  errMessage := "There was an error processing the query: %s"
  // Check that the query isn't too large.
  if len(message.Content) > queryLenMax {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Please keep queries below %d characters.", queryLenMax))
    return
  }
  // Check for a "Fresh:" field which may interfere.
  if strings.Contains(message.Content, "Fresh:") {
    session.ChannelMessageSend(message.ChannelID, "Query cannot contain the field \"Fresh\".")
    return
  }
  // Check for regex-delimitting forward-slashes.
  if strings.ContainsRune(message.Content, '/') {
    session.ChannelMessageSend(message.ChannelID, "Query cannoth contain the forward-slash character (\"/\").")
    return
  }
  s, err := translateQuery(message.Content[len(env.Config.Prefix)+len("lookout"):])
  if err != nil {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, err.Error()))
    return
  }
  // Check that the server is specified, and matches an existing server.
  re := regexp.MustCompile(`\+?Server:\s*(\w+)`)
  sMatches := re.FindStringSubmatch(s)
  if (sMatches != nil) {
    sMatch := sMatches[1]
    env.AuditLock.RLock()
    _, exists := env.Audit.Map[sMatch]
    env.AuditLock.RUnlock()
    if !exists {
      session.ChannelMessageSend(message.ChannelID, "The requested query does not seem to specify an existing server.")
      return
    }
    s = re.ReplaceAllString(s, "+Server: $1")
  } else {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "Missing a server field."))
    return
  }
  // Verify and parse out the duration field.
  durRegex := regexp.MustCompile(`\s+Duration:\s*(\S+)`)
  matchSlice := durRegex.FindStringSubmatch(s)
  if len(matchSlice) != 2 {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "Unable to locate a duration field."))
    return
  }
  dur, err := time.ParseDuration(matchSlice[1])
  if err != nil {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "Unable to parse the duration."))
    return
  }
  // Verify duration is within acceptable range.
  if dur > (time.Hour * 24) {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "The duration is longer than twenty-four hours."))
    return
  }
  if dur < time.Nanosecond {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "The duration seems awfully small."))
    return
  }
  // Finally, remove the duration field from the query.
  s = strings.Replace(s, matchSlice[0], "", 1)

  var q lodb.LoQuery
  q = lodb.LoQuery{
    AuthorID: message.Author.ID,
    ChannelID: message.ChannelID,
    TTL: dur,
    Query: s,
  }
  // Save query to the repository.
  env.TickLock.RLock()
  t := env.Tick
  errS := env.Repo.Save(q, t)
  env.TickLock.RUnlock()
  if errS != nil {
    env.Log.Error(errS)
    session.ChannelMessageSend(q.ChannelID, "Oh dear, it seems like there was a problem.")
  } else {
    env.Log.Info("New query.")
    session.ChannelMessageSend(q.ChannelID, "Lookout query saved.")
  }
  env.Log.Trace("Lookout command processed.")
}

// Translates our slightly more friendly format into a valid Bleve string query.
func translateQuery(s string) (string, error) {
  return replaceLevel(s)
  // TODO: Make sure Server field has Title-case.
}

func replaceLevel(s string) (string, error) {
  splits := regexp.MustCompile(`\s*Level:`).Split(s, 3)
  // Return the string if no level field found.
  if len(splits) < 2 {
    return s, nil
  } else if len(splits) > 2 {
    // If unexpected splits are encountered, return error on multiple level fields.
    return s, errors.New("Multiple fields found specifying level.")
  } else {
    // If a level field is found, try parsing an integer from it.
    fields := strings.Fields(splits[1])
    num, err := strconv.ParseInt(fields[0], 10, 8)
    if err != nil {
      return s, errors.New("Problem parsing integer for level field.")
    }
    if num < 1 {
      return s, errors.New("Non-positive integer parsed from level field.")
    }
    after := strings.Join( fields[1:], " ")
    return fmt.Sprintf("%s +Group.MaxLevel:>=%d +Group.MinLevel:<=%d %s", splits[0], num, num, after), nil
  }
}

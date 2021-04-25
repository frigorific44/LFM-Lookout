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
  errMessage := "There was an error processing the query: %s"
  s, err := TranslateQuery(message.Content[len(env.Config.Prefix)+len("lookout"):])
  if err != nil {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, err.Error()))
    return
  }
  // Verify the existence of a server field.
  if !strings.Contains(s, "Server:") {
    session.ChannelMessageSend(message.ChannelID, fmt.Sprintf(errMessage, "Missing a server field."))
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
  env.LoChan <- q
}

// Translates our slightly more friendly format into a valid Bleve string query.
func TranslateQuery(s string) (string, error) {
  return ReplaceLevel(s)
  // TODO: Make sure Server field has Title-case.
}

func ReplaceLevel(s string) (string, error) {
  splits := regexp.MustCompile(`\s*level:`).Split(s, 3)
  // Return the string if no level field found.
  if len(splits) < 2 {
    return s, nil
  // If a level field is found, try parsing an integer from it.
  } else if len(splits) > 2 {
    return s, errors.New("Multiple fields found specifying level.")
  } else {
    fields := strings.Fields(splits[1])
    num, err := strconv.ParseInt(fields[0], 10, 8)
    if err != nil {
      return s, errors.New("Problem parsing integer for level field.")
    }
    if num < 1 {
      return s, errors.New("Non-positive integer parsed from level field.")
    }
    after := strings.Join( fields[:], " ")
    return fmt.Sprintf("%s maxlevel:>=%d minlevel:<=%d %s", splits[0], num, num, after), nil
  // If unexpected splits are encountered, return error on multiple level fields.
  }
}

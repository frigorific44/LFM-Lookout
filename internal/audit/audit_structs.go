package audit

import (
  "fmt"
  "strings"
)

// Below are the structs which GET call to PlayerAudit.com will be
// unmarshalled into.

// A full representation of the response has been stemmed out, but a selection
// of fields have been commented out, as having been deemed unnecessary for
// our purposes here.

type Audit struct {
  Servers []Server
}

type Server struct {
  Name string `json:"Name"`
  // LastUpdate string `json:"LastUpdateTime"`
  // GroupCount int `json:"GroupCount"`
  Groups []Group `json:"Groups"`
}

type Group struct {
  Id uint64 `json:"Id"`
  Comment string `json:"Comment"`
  Quest Quest `json:"Quest"`
  Difficulty string `json:"Difficulty"`
  // AcceptedClasses []string `json:"AcceptedClasses"`
  // AcceptedCount uint8 `json:"AcceptedCount"`
  MinLevel int `json:"MinimumLevel"`
  MaxLevel int `json:"MaximumLevel"`
  AdventureActive uint8 `json:"AdventureActive"`
  Leader Member `json:"Leader"`
  Members []Member `json:"Members"`
}

type Quest struct {
  Name string `json:"Name"`
  // CRHeroic int `json:"CR_Heroic"`
  // CREpic int `json:"CR_Epic"`
  Patron string `json:"Patron"`
  Type string `json:"Type"`
}

type Member struct {
  Location Location `json:"Location"`
  // Name string `json:"Name"`
  // Gender string `json:"Gender"`
  // Race string `json:"Race"`
  // TotalLevel uint8 `json:"TotalLevel"`
  // Classes []Class `json:"Classes"`
}

// type Class struct {
//   Name string `json:"Name"`
//   Level uint8 `json:"Level"`
// }

type Location struct {
  Name string `json:"Name"`
  // IsPublicSpace bool `json:"IsPublicSpace"`
}

func (group Group) String() string {
  lines := []string{}
  // Line of quest info, if there is a quest selected.
  if group.Quest != (Quest{}) {
    lines = append(lines, group.Quest.String())
  }
  // Line of group members and their locations.
  numMembers := 1 + len(group.Members)
  lines = append(lines, fmt.Sprintf("%d Member(s)", numMembers))
  // Line of general info: Comment, Adventure Active, Difficulty, Level Range
  info := []string{}
  if group.Comment != "" {
    info = append(info, group.Comment)
  }
  if group.AdventureActive != 0 {
    active := fmt.Sprintf("Active: %d minute(s)", group.AdventureActive)
    info = append(info, active)
  }
  info = append(info, group.Difficulty)
  lvlRange := fmt.Sprintf("[%d–%d]", group.MinLevel, group.MaxLevel)
  info = append(info, lvlRange)
  lines = append(lines, strings.Join(info, " | "))

  return strings.Join(lines, "\n")
}

func (quest Quest) String() string {
  s := fmt.Sprintf("%s: %s, %s", quest.Name, quest.Type, quest.Patron)
  return s
}

func (member Member) String() string {
  s := member.Location.Name
  return s
}

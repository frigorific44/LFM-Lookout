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
	Id         uint64 `json:"Id"`
	Comment    string `json:"Comment"`
	Quest      Quest  `json:"Quest"`
	Difficulty string `json:"Difficulty"`
	// AcceptedClasses []string `json:"AcceptedClasses"`
	// AcceptedCount uint8 `json:"AcceptedCount"`
	MinLevel        int      `json:"MinimumLevel"`
	MaxLevel        int      `json:"MaximumLevel"`
	AdventureActive uint8    `json:"AdventureActive"`
	Leader          Member   `json:"Leader"`
	Members         []Member `json:"Members"`
}

type Quest struct {
	Name string `json:"Name"`
	// HeroicNormalCR uint8 `json:"HeroicNormalCR"`
	// EpicNormalCR uint8 `json:"EpicNormalCR"`
	// HeroicNormalXp int `json:"HeroicNormalXp"`
	// HeroicHardXp int `json:"HeroicHardXp"`
	// HeroicEliteXp int `json:"HeroicEliteXp"`
	// EpicNormalXp int `json:"EpicNormalXp"`
	// EpicHardXp int `json:"EpicHardXp"`
	// EpicEliteXp int `json:"EpicEliteXp"`
	// IsFreeToVip bool `json:"IsFreeToVip"`
	AdventurePack     string `json:"RequiredAdventurePack"`
	AdventureArea     string `json:"AdventureArea"`
	QuestJournalGroup string `json:"QuestJournalGroup"`
	GroupSize         string `json:"GroupSize"`
	Patron            string `json:"Patron"`
}

type Member struct {
	Location Location `json:"Location"`
	// Name string `json:"Name"`
	// Gender string `json:"Gender"`
	// Race string `json:"Race"`
	// TotalLevel uint8 `json:"TotalLevel"`
	// Classes []Class `json:"Classes"`
	// GroupId uint64 `json:"GroupId"`
	// Guild string `json:"Guild"`
	// InParty bool `json:"InParty"`
	// HomeServer string `json:"HomeServer"`
}

// type Class struct {
//   Name string `json:"Name"`
//   Level uint8 `json:"Level"`
// }

type Location struct {
	Name string `json:"Name"`
	// IsPublicSpace bool `json:"IsPublicSpace"`
	Region string `json:"Region"`
}

func (group Group) String() string {
	b := strings.Builder{}
	// Line of quest info, and how long it has been active.
	if group.Quest != (Quest{}) {
		fmt.Fprintf(&b, "> %s, %s", group.Quest.Name, group.Quest.Patron)
		if group.AdventureActive != 0 {
			fmt.Fprintf(&b, " | *Active: %d minute(s)*", group.AdventureActive)
		}
		b.WriteString("\n")
	}
	// Line with level range, number of members, and difficulty.
	numMembers := 1 + len(group.Members)
	fmt.Fprintf(&b, "> **%d-%d** | %d Member(s) | %s", group.MinLevel, group.MaxLevel, numMembers, group.Difficulty)
	// Line with comment.
	if group.Comment != "" {
		fmt.Fprintf(&b, "\n> %s", group.Comment)
	}

	return b.String()
}

package botenv

import (
	"lfm_lookout/internal/audit"
	"lfm_lookout/internal/lodb"

	"sync"

	"go.uber.org/zap"
)

type SearchableGroup struct {
	Server  string
	Group   audit.Group
	Members int
	Fresh   bool
}

type AuditMap struct {
	Map map[string]map[string]SearchableGroup
}

type BotEnv struct {
	Config *Configuration
	Log    *zap.Logger
	Repo   *lodb.LoRepo
	// map[audit.Server.Name]map["audit.Group.Id"]audit.Group
	Audit     AuditMap
	AuditLock *sync.RWMutex
	Tick      rune
	TickLock  *sync.RWMutex
}

type Configuration struct {
	Prefix string `json:"Prefix"`
	Token  string `json:"Token"`
	AuditPeriod int `json:"AuditPeriod"`
}

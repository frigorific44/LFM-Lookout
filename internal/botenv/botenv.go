package botenv

import (
  "lfm_lookout/internal/audit"
  "lfm_lookout/internal/lodb"

  "sync"

  logrus "github.com/sirupsen/logrus"
)

type SearchableGroup struct {
	Server string
	Group audit.Group
  Fresh bool
}

type AuditMap struct {
  Map map[string]map[string]SearchableGroup
}

type BotEnv struct {
  Config *Configuration
  Log *logrus.Logger
  Repo *lodb.LoRepo
  // map[audit.Server.Name]map["audit.Group.Id"]audit.Group
  Audit AuditMap
  AuditLock *sync.RWMutex
  Tick rune
  TickLock *sync.RWMutex
}

type Configuration struct {
  Prefix string `json:"Prefix"`
  Token string `json:"Token"`
}

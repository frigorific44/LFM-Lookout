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
}

type BotEnv struct {
  Config *Configuration
  Log *logrus.Logger
  Repo *lodb.LoRepo
  // map[audit.Server.Name]map["audit.Group.Id"]audit.Group
  Audit map[string]map[string]SearchableGroup
  AuditLock *sync.RWMutex
  LoChan chan lodb.LoQuery
}

type Configuration struct {
  Prefix string `json:"Prefix"`
  Token string `json:"Token"`
}

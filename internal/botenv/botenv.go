package botenv

import (
  "lfm_lookout/internal/audit"
  "lfm_lookout/internal/lodb"

  "sync"

  logrus "github.com/sirupsen/logrus"
)

type BotEnv struct {
  Config *Configuration
  Log *logrus.Logger
  Repo *lodb.LoRepo
  Audit *audit.Audit
  AuditLock *sync.RWMutex
}

type Configuration struct {
  Prefix string `json:"Prefix"`
  Token string `json:"Token"`
}

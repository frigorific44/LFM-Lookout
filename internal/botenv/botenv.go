package botenv

import (
  "lfm_lookout/internal/audit"
  "lfm_lookout/internal/lodb"

  logrus "github.com/sirupsen/logrus"
)

type BotEnv struct {
  Config Configuration
  Log *logrus.Logger
  Repo *lodb.LoRepo
  Audit *audit.Audit
}

type Configuration struct {
  Prefix string `json:"Prefix"`
  Token string `json:"Token"`
}

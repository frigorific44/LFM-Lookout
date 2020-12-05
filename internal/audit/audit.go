package audit

import (
  "encoding/json"
  "io/ioutil"
  "net/http"
)


func Groups() (*Audit, error) {
  var auditObject Audit

  const groupsURL = "https://www.playeraudit.com/api/groups"
  response, err := http.Get(groupsURL)
  if err != nil {
    return &auditObject, err
  }

  auditData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return &auditObject, err
  }

  json.Unmarshal(auditData, &auditObject.Servers)

  return &auditObject, nil
}

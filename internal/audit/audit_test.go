package audit_test

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "testing"

  "lfm_lookout/internal/audit"
)


func TestGroups(t *testing.T) {
  resp, err := audit.Groups()
  if err != nil {
    t.Error(err)
  }
  if len(resp.Servers) <= 0 {
    t.Error("No content.")
  }
  respJSON, _ := json.Marshal(resp)
  if len(string(respJSON)) < 200 {
    t.Error("JSON appears too small.")
  }
}

func TestGroup_String(t *testing.T) {
  var a audit.Audit
  // Load the config.json file.
  file, err := ioutil.ReadFile("groups.json")
  if err != nil {
    t.Error(err)
  }
  // Load JSON into botenv:config.
  // fmt.Println(string(file))
  json.Unmarshal(file, &a.Servers)

  for _, g := range a.Servers[0].Groups {
    fmt.Println(g.String())
    fmt.Println("")
  }
}

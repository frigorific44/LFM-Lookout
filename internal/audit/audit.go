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

// More precisely, this is checking for a significant
// enough change in state to warrant re-checking it against queries, as it will
// only be used between different time-points of the same group ID.
func (g1 *Group) Changed(g2 Group) bool {
	switch {
	case g1.Comment != g2.Comment:
		return true
	case g1.Quest.Name != g2.Quest.Name:
		return true
	case g1.Difficulty != g2.Difficulty:
		return true
	case g1.MinLevel != g2.MinLevel:
		return true
	case g1.MaxLevel != g2.MaxLevel:
		return true
	case g1.AdventureActive > uint8(0) && g2.AdventureActive == uint8(0):
		return true
	default:
		return false
	}
}

package intents

import (
	"strings"
)

type IntentInfo struct {
	Name      string
	Intent    string
	BitOffset int
	Events    []string
	DM        bool
}

func (i IntentInfo) String() string {
	return i.Name
}

func (i IntentInfo) IsDM() bool {
	return strings.HasPrefix(i.Name, "DirectMessage") || i.DM
}

package encoding

import "encoding/json"

var (
	Marshal   = json.Marshal
	Unmarshal = json.Unmarshal
)

type (
	RawMessage = json.RawMessage
)

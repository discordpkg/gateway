package json

import (
	"encoding/json"
)

// ensure that the interfaces are the same as in the json pkg
type custom struct {}

func (c custom) UnmarshalJSON(_ []byte) error {
	return nil
}

func (c custom) MarshalJSON() ([]byte, error) {
	return nil, nil
}

var _ Unmarshaler = &custom{}
var _ json.Unmarshaler = &custom{}
var _ Marshaler = &custom{}
var _ json.Marshaler = &custom{}
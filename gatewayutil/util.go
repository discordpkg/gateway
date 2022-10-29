package gatewayutil

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/discordpkg/gateway/json"

	"github.com/discordpkg/gateway"
)

func DeriveShardID(snowflake uint64, totalNumberOfShards uint) gateway.ShardID {
	createdUnix := snowflake >> 22
	groups := uint64(totalNumberOfShards)
	return gateway.ShardID(createdUnix % groups)
}

var supportedAPIVersions = []string{
	"8", "9", "10",
}
var supportedAPICodes = []string{
	"json",
}

var ErrURLScheme = errors.New("url scheme was not websocket (ws nor wss)")
var ErrUnsupportedAPIVersion = fmt.Errorf("only discord api version %+v is supported", supportedAPIVersions)
var ErrUnsupportedAPICodec = fmt.Errorf("only %+v is supported", supportedAPICodes)
var ErrIncompleteDialURL = errors.New("incomplete url is missing one or many of: 'version', 'encoding', 'scheme'")

func ValidateDialURL(URLString string) (string, error) {
	in := func(keyword string, slice []string) bool {
		for i := range slice {
			if keyword == slice[i] {
				return true
			}
		}
		return false
	}

	u, urlErr := url.Parse(URLString)
	if urlErr != nil {
		return "", urlErr
	}

	scheme := u.Scheme
	v := u.Query().Get("v")
	encoding := u.Query().Get("encoding")

	if v == "" || encoding == "" || scheme == "" {
		return "", ErrIncompleteDialURL
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return "", ErrURLScheme
	}
	if v != "" && !in(v, supportedAPIVersions) {
		return "", ErrUnsupportedAPIVersion
	}
	if encoding != "" && !in(encoding, supportedAPICodes) {
		return "", ErrUnsupportedAPICodec
	}

	return u.String(), nil
}

func ReadPayload(client io.Reader) (*gateway.Payload, int, error) {
	data, err := io.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &gateway.Payload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	return packet, len(data), nil
}

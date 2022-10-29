package gatewayutil

import (
	"testing"

	"github.com/discordpkg/gateway"
)

func TestClientShard(t *testing.T) {
	t.Run("one-gatewayutil", func(t *testing.T) {
		randomSnowflakes := []uint64{
			345573676574567,
			47890435843,
			23940234,
			2987509435,
			94385743905733,
			453876485923485,
			5487365834,
			1345987340925,
		}

		for _, s := range randomSnowflakes {
			if DeriveShardID(s, 1) != 0 {
				t.Errorf("expected gatewayutil id to be 0, got %d", s)
			}
		}
	})
	t.Run("multiple-shards", func(t *testing.T) {
		shift := func(s uint64) uint64 {
			return s << 22
		}
		snowflakes := []int{0, 0, 0, 0, 0, 0}

		for i := range snowflakes {
			s := shift(uint64(i))
			shardID := DeriveShardID(s, uint(len(snowflakes)))
			if shardID != gateway.ShardID(i) {
				t.Errorf("expected gatewayutil id to be %d, got %d", i, shardID)
			}
		}
	})
}

func TestValidateDialURL(t *testing.T) {

	type table struct {
		name          string
		urlString     string
		expectedError error
	}

	tests := []table{
		{"wrong schema", "http://gateway.discord.gg/?v=10&encoding=json", ErrURLScheme},
		{"wrong schema", "https://gateway.discord.gg/?v=10&encoding=json", ErrURLScheme},
		{"incomplete url", "wss://gateway.discord.gg/", ErrIncompleteDialURL},
		{"incomplete url", "wss://gateway.discord.gg/", ErrIncompleteDialURL},
		{"incomplete url", "wss://gateway.discord.gg/?v=1", ErrIncompleteDialURL},
		{"incomplete url", "wss://gateway.discord.gg/?encoding=json", ErrIncompleteDialURL},
		{"old api version", "wss://gateway.discord.gg/?v=1&encoding=json", ErrUnsupportedAPIVersion},
		{"wrong encoding", "wss://gateway.discord.gg/?v=10&encoding=mysql", ErrUnsupportedAPICodec},
		{"valid url", "wss://gateway.discord.gg/?v=10&encoding=json", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateDialURL(test.urlString)
			if err != test.expectedError {
				t.Errorf("got error '%+v', expected '%+v'", err, test.expectedError)
			}
		})
	}
}

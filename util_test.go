package discordgateway

import (
	"testing"
)

func TestClientShard(t *testing.T) {
	t.Run("one-shard", func(t *testing.T) {
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
				t.Errorf("expected shard id to be 0, got %d", s)
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
			if shardID != ShardID(i) {
				t.Errorf("expected shard id to be %d, got %d", i, shardID)
			}
		}
	})
}

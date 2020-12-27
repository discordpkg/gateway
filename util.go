package discordgateway

func ClientShard(guildID uint64, totalNumberOfShards uint) (shardID uint) {
	createdUnix := guildID >> 22
	groups := uint64(totalNumberOfShards)
	return uint(createdUnix % groups)
}

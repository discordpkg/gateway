package discordgateway

func DeriveShardID(snowflake uint64, totalNumberOfShards uint) ShardID {
	createdUnix := snowflake >> 22
	groups := uint64(totalNumberOfShards)
	return ShardID(createdUnix % groups)
}

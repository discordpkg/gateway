package closecode

type Type uint32

// custom codes
const (
	HeartbeatAckNotReceived Type = 3000
	ClientReconnecting      Type = 1001
)

const (
	// UnknownError We're not sure what went wrong. Try reconnecting?
	UnknownError Type = 4000 + iota
	// UnknownOpCode You sent an invalid Gateway opcode or an invalid payload for an opcode. Don't do that!
	UnknownOpCode
	// DecodeError You sent an invalid payload to us. Don't do that!
	DecodeError
	// NotAuthenticated You sent us a payload prior to identifying
	NotAuthenticated
	// AuthenticationFailed The account token sent with your identify payload is incorrect
	AuthenticationFailed
	// AlreadyAuthenticated You sent more than one identify payload. Don't do that!
	AlreadyAuthenticated
	_ // 4006
	// InvalidSeq The sequence sent when resuming the session was invalid. Reconnect and start a new session
	InvalidSeq
	// RateLimited Woah nelly! You're sending payloads to us too quickly. Slow it down! You will be disconnected on receiving this
	RateLimited
	// SessionTimedOut Your session timed out. Reconnect and start a new one
	SessionTimedOut
	// InvalidShard You sent us an invalid shard when identifying
	InvalidShard
	// ShardingRequired The session would have handled too many guilds - you are required to shard your connection in order to connect
	ShardingRequired
	// InvalidAPIVersion You sent an invalid version for the gateway
	InvalidAPIVersion
	// InvalidIntents You sent an invalid intent for a Gateway Intent. You may have incorrectly calculated the bitwise value
	InvalidIntents
	// DisallowedIntents You sent a disallowed intent for a Gateway Intent. You may have tried to specify an intent that you have not enabled or are not whitelisted for
	DisallowedIntents
)

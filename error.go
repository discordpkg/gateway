package discordgateway

//go:generate go run internal/generate/errors/main.go

type DiscordErrorCode uint32

// custom codes
const (
	HeartbeatAckNotReceived DiscordErrorCode = 3000
)

// ////////////////////////////////////////////////////
//
// GATEWAY: error codes and types
//
// ////////////////////////////////////////////////////

type CloseCode DiscordErrorCode

const (
	// CloseCodeUnknownError We're not sure what went wrong. Try reconnecting?
	CloseCodeUnknownError CloseCode = 4000 + iota
	// CloseCodeUnknownOpCode You sent an invalid Gateway opcode or an invalid payload for an opcode. Don't do that!
	CloseCodeUnknownOpCode
	// CloseCodeDecodeError You sent an invalid payload to us. Don't do that!
	CloseCodeDecodeError
	// CloseCodeNotAuthenticated You sent us a payload prior to identifying
	CloseCodeNotAuthenticated
	// CloseCodeAuthenticationFailed The account token sent with your identify payload is incorrect
	CloseCodeAuthenticationFailed
	// CloseCodeAlreadyAuthenticated You sent more than one identify payload. Don't do that!
	CloseCodeAlreadyAuthenticated
	_ // 4006
	// CloseCodeInvalidSeq The sequence sent when resuming the session was invalid. Reconnect and start a new session
	CloseCodeInvalidSeq
	// CloseCodeRateLimited Woah nelly! You're sending payloads to us too quickly. Slow it down! You will be disconnected on receiving this
	CloseCodeRateLimited
	// CloseCodeSessionTimedOut Your session timed out. Reconnect and start a new one
	CloseCodeSessionTimedOut
	// CloseCodeInvalidShard You sent us an invalid shard when identifying
	CloseCodeInvalidShard
	// CloseCodeShardingRequired The session would have handled too many guilds - you are required to shard your connection in order to connect
	CloseCodeShardingRequired
	// CloseCodeInvalidAPIVersion You sent an invalid version for the gateway
	CloseCodeInvalidAPIVersion
	// CloseCodeInvalidIntents You sent an invalid intent for a Gateway Intent. You may have incorrectly calculated the bitwise value
	CloseCodeInvalidIntents
	// CloseCodeDisallowedIntents You sent a disallowed intent for a Gateway Intent. You may have tried to specify an intent that you have not enabled or are not whitelisted for
	CloseCodeDisallowedIntents
)

type CloseErr struct {
	Code CloseCode
	Err  error
}

var _ error = (*CloseErr)(nil)

// ////////////////////////////////////////////////////
//
// VOICE: error codes and types
//
// ////////////////////////////////////////////////////

type VoiceCloseEventCode DiscordErrorCode

const (
	_ VoiceCloseEventCode = 4000 + iota
	// VoiceCloseCodeUnknownOpCode You sent an invalid opcode
	VoiceCloseCodeUnknownOpCode
	_
	// VoiceCloseCodeNotAuthenticated You sent a payload before identifying with the Gateway
	VoiceCloseCodeNotAuthenticated
	// VoiceCloseCodeAuthenticationFailed The token you sent in your identify payload is incorrect
	VoiceCloseCodeAuthenticationFailed
	// VoiceCloseCodeAlreadyAuthenticated You sent more than one identify payload. Stahp
	VoiceCloseCodeAlreadyAuthenticated
	// VoiceCloseCodeSessionNoLongerValid Your session is no longer valid
	VoiceCloseCodeSessionNoLongerValid
	_ // 4007
	_ // 4008
	// VoiceCloseCodeSessionTimedOut Your session has timed out
	VoiceCloseCodeSessionTimedOut
	_ // 4010
	// VoiceCloseCodeServerNotFound We can't find the server you're trying to connect to
	VoiceCloseCodeServerNotFound
	// VoiceCloseCodeUnknownProtocol We didn't recognize the protocol you sent
	VoiceCloseCodeUnknownProtocol
	_ // 4013
	// VoiceCloseCodeDisconnected Either the channel was deleted or you were kicked. Should not reconnect
	VoiceCloseCodeDisconnected
	// VoiceCloseCodeVoiceServerCrashed The server crashed. Our bad! Try resuming
	VoiceCloseCodeVoiceServerCrashed
	// VoiceCloseCodeUnknownEncryptionMode We didn't recognize your encryption
	VoiceCloseCodeUnknownEncryptionMode
)

type DiscordVoiceCloseErr struct {
	Code VoiceCloseEventCode
}

var _ error = (*DiscordVoiceCloseErr)(nil)

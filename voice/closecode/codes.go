package closecode

import "github.com/andersfylling/discordgateway/closecode"

const (
	_ closecode.Type = 4000 + iota
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

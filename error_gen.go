package discordgateway

// Code generated - This file has been automatically generated by internal/generate/errors/main.go - DO NOT EDIT.
// Warning: This file is overwritten at "go generate", instead adapt error.go and run go generate
const (
	VoiceCloseCodeUnknownOpCodeReason         = "VoiceCloseCodeUnknownOpCode You sent an invalid opcode"
	VoiceCloseCodeNotAuthenticatedReason      = "VoiceCloseCodeNotAuthenticated You sent a payload before identifying with the Gateway"
	VoiceCloseCodeAuthenticationFailedReason  = "VoiceCloseCodeAuthenticationFailed The token you sent in your identify payload is incorrect"
	VoiceCloseCodeAlreadyAuthenticatedReason  = "VoiceCloseCodeAlreadyAuthenticated You sent more than one identify payload. Stahp"
	VoiceCloseCodeSessionNoLongerValidReason  = "VoiceCloseCodeSessionNoLongerValid Your session is no longer valid"
	VoiceCloseCodeSessionTimedOutReason       = "VoiceCloseCodeSessionTimedOut Your session has timed out"
	VoiceCloseCodeServerNotFoundReason        = "VoiceCloseCodeServerNotFound We can't find the server you're trying to connect to"
	VoiceCloseCodeUnknownProtocolReason       = "VoiceCloseCodeUnknownProtocol We didn't recognize the protocol you sent"
	VoiceCloseCodeDisconnectedReason          = "VoiceCloseCodeDisconnected Either the channel was deleted or you were kicked. Should not reconnect"
	VoiceCloseCodeVoiceServerCrashedReason    = "VoiceCloseCodeVoiceServerCrashed The server crashed. Our bad! Try resuming"
	VoiceCloseCodeUnknownEncryptionModeReason = "VoiceCloseCodeUnknownEncryptionMode We didn't recognize your encryption"
)

func (e *DiscordVoiceCloseErr) Error() string {
	var msg string
	switch e.Code {
	case VoiceCloseCodeUnknownOpCode:
		msg = VoiceCloseCodeUnknownOpCodeReason
	case VoiceCloseCodeNotAuthenticated:
		msg = VoiceCloseCodeNotAuthenticatedReason
	case VoiceCloseCodeAuthenticationFailed:
		msg = VoiceCloseCodeAuthenticationFailedReason
	case VoiceCloseCodeAlreadyAuthenticated:
		msg = VoiceCloseCodeAlreadyAuthenticatedReason
	case VoiceCloseCodeSessionNoLongerValid:
		msg = VoiceCloseCodeSessionNoLongerValidReason
	case VoiceCloseCodeSessionTimedOut:
		msg = VoiceCloseCodeSessionTimedOutReason
	case VoiceCloseCodeServerNotFound:
		msg = VoiceCloseCodeServerNotFoundReason
	case VoiceCloseCodeUnknownProtocol:
		msg = VoiceCloseCodeUnknownProtocolReason
	case VoiceCloseCodeDisconnected:
		msg = VoiceCloseCodeDisconnectedReason
	case VoiceCloseCodeVoiceServerCrashed:
		msg = VoiceCloseCodeVoiceServerCrashedReason
	case VoiceCloseCodeUnknownEncryptionMode:
		msg = VoiceCloseCodeUnknownEncryptionModeReason
	}
	return msg
}

const (
	CloseCodeUnknownErrorReason         = "CloseCodeUnknownError We're not sure what went wrong. Try reconnecting?"
	CloseCodeUnknownOpCodeReason        = "CloseCodeUnknownOpCode You sent an invalid Gateway opcode or an invalid payload for an opcode. Don't do that!"
	CloseCodeDecodeErrorReason          = "CloseCodeDecodeError You sent an invalid payload to us. Don't do that!"
	CloseCodeNotAuthenticatedReason     = "CloseCodeNotAuthenticated You sent us a payload prior to identifying"
	CloseCodeAuthenticationFailedReason = "CloseCodeAuthenticationFailed The account token sent with your identify payload is incorrect"
	CloseCodeAlreadyAuthenticatedReason = "CloseCodeAlreadyAuthenticated You sent more than one identify payload. Don't do that!"
	CloseCodeInvalidSeqReason           = "CloseCodeInvalidSeq The sequence sent when resuming the session was invalid. Reconnect and start a new session"
	CloseCodeRateLimitedReason          = "CloseCodeRateLimited Woah nelly! You're sending payloads to us too quickly. Slow it down! You will be disconnected on receiving this"
	CloseCodeSessionTimedOutReason      = "CloseCodeSessionTimedOut Your session timed out. Reconnect and start a new one"
	CloseCodeInvalidShardReason         = "CloseCodeInvalidShard You sent us an invalid shard when identifying"
	CloseCodeShardingRequiredReason     = "CloseCodeShardingRequired The session would have handled too many guilds - you are required to shard your connection in order to connect"
	CloseCodeInvalidAPIVersionReason    = "CloseCodeInvalidAPIVersion You sent an invalid version for the gateway"
	CloseCodeInvalidIntentsReason       = "CloseCodeInvalidIntents You sent an invalid intent for a Gateway Intent. You may have incorrectly calculated the bitwise value"
	CloseCodeDisallowedIntentsReason    = "CloseCodeDisallowedIntents You sent a disallowed intent for a Gateway Intent. You may have tried to specify an intent that you have not enabled or are not whitelisted for"
)

func (e *CloseErr) Error() string {
	var msg string
	switch e.Code {
	case CloseCodeUnknownError:
		msg = CloseCodeUnknownErrorReason
	case CloseCodeUnknownOpCode:
		msg = CloseCodeUnknownOpCodeReason
	case CloseCodeDecodeError:
		msg = CloseCodeDecodeErrorReason
	case CloseCodeNotAuthenticated:
		msg = CloseCodeNotAuthenticatedReason
	case CloseCodeAuthenticationFailed:
		msg = CloseCodeAuthenticationFailedReason
	case CloseCodeAlreadyAuthenticated:
		msg = CloseCodeAlreadyAuthenticatedReason
	case CloseCodeInvalidSeq:
		msg = CloseCodeInvalidSeqReason
	case CloseCodeRateLimited:
		msg = CloseCodeRateLimitedReason
	case CloseCodeSessionTimedOut:
		msg = CloseCodeSessionTimedOutReason
	case CloseCodeInvalidShard:
		msg = CloseCodeInvalidShardReason
	case CloseCodeShardingRequired:
		msg = CloseCodeShardingRequiredReason
	case CloseCodeInvalidAPIVersion:
		msg = CloseCodeInvalidAPIVersionReason
	case CloseCodeInvalidIntents:
		msg = CloseCodeInvalidIntentsReason
	case CloseCodeDisallowedIntents:
		msg = CloseCodeDisallowedIntentsReason
	}
	return msg
}

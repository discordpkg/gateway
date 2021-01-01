package opcode

const (
	VoiceIdentify OpCode = iota
	VoiceSelectProtocol
	VoiceReady
	VoiceHeartbeat
	VoiceSessionDescription
	VoiceSpeaking
	VoiceHeartbeatAck
	VoiceResume
	VoiceHello
	VoiceResumed
	_
	_
	_
	VoiceClientDisconnect
)

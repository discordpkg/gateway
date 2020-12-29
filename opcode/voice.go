package opcode

const (
	VoiceIdentify           OpCode = 0 | internalOnly | voice | send
	VoiceSelectProtocol     OpCode = 1 | internalOnly | voice | send
	VoiceReady              OpCode = 2 | internalOnly | voice | receive
	VoiceHeartbeat          OpCode = 3 | internalOnly | voice | send
	VoiceSessionDescription OpCode = 4 | internalOnly | voice | receive
	VoiceSpeaking           OpCode = 5 | internalOnly | voice | send | receive
	VoiceHeartbeatAck       OpCode = 6 | internalOnly | voice | receive
	VoiceResume             OpCode = 7 | internalOnly | voice | send
	VoiceHello              OpCode = 8 | internalOnly | voice | receive
	VoiceResumed            OpCode = 9 | internalOnly | voice | receive
	VoiceClientDisconnect   OpCode = 13 | internalOnly | voice | receive
)

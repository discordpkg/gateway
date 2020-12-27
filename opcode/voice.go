package opcode

const (
	VoiceIdentify           Op = 0 | internalOnly | send
	VoiceSelectProtocol     Op = 1 | internalOnly | send
	VoiceReady              Op = 2 | internalOnly | receive
	VoiceHeartbeat          Op = 3 | internalOnly | send
	VoiceSessionDescription Op = 4 | internalOnly | receive
	VoiceSpeaking           Op = 5 | internalOnly | send | receive
	VoiceHeartbeatAck       Op = 6 | internalOnly | receive
	VoiceResume             Op = 7 | internalOnly | send
	VoiceHello              Op = 8 | internalOnly | receive
	VoiceResumed            Op = 9 | internalOnly | receive
	VoiceClientDisconnect   Op = 13 | internalOnly | receive
)

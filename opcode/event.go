package opcode

const (
	EventDispatch OpCode = iota
	EventHeartbeat
	EventIdentify
	EventPresenceUpdate
	EventVoiceStateUpdate
	_
	EventResume
	EventReconnect
	EventRequestGuildMembers
	EventInvalidSession
	EventHello
	EventHeartbeatACK
)

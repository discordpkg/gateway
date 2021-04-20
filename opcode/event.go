package opcode

// send / receive
const (
	EventHeartbeat OpCode = 1
	_
)

// send only
const (
	EventIdentify            OpCode = 2
	EventPresenceUpdate      OpCode = 3
	EventVoiceStateUpdate    OpCode = 4
	EventResume              OpCode = 6
	EventRequestGuildMembers OpCode = 8
)

// receive only
const (
	EventDispatch       OpCode = 0
	EventReconnect      OpCode = 7
	EventInvalidSession OpCode = 9
	EventHello          OpCode = 10
	EventHeartbeatACK   OpCode = 11
)

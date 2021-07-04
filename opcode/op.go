package opcode

// Type is the discord operation value
type Type uint

const (
	Invalid Type = 0b111111111111 // 4095
)

// send / receive
const (
	Heartbeat Type = 1
	_
)

// send only
const (
	Identify            Type = 2
	PresenceUpdate      Type = 3
	VoiceStateUpdate    Type = 4
	Resume              Type = 6
	RequestGuildMembers Type = 8
)

// receive only
const (
	Dispatch       Type = 0
	Reconnect      Type = 7
	InvalidSession Type = 9
	Hello          Type = 10
	HeartbeatACK   Type = 11
)

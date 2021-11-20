package opcode

import "github.com/andersfylling/discordgateway/command"

// Type is the discord operation value
type Type uint

const (
	Invalid Type = 0b111111111111 // 4095
)

// send / receive
const (
	Heartbeat = Type(command.Heartbeat)
	_
)

// send only
const (
	Identify            = Type(command.Identify)
	PresenceUpdate      = Type(command.UpdatePresence)
	VoiceStateUpdate    = Type(command.UpdateVoiceState)
	Resume              = Type(command.Resume)
	RequestGuildMembers = Type(command.RequestGuildMembers)
)

// receive only
const (
	Dispatch       Type = 0
	Reconnect      Type = 7
	InvalidSession Type = 9
	Hello          Type = 10
	HeartbeatACK   Type = 11
)

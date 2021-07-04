package opcode

import "github.com/andersfylling/discordgateway/opcode"

const (
	Identify opcode.Type = iota
	SelectProtocol
	Ready
	Heartbeat
	SessionDescription
	Speaking
	HeartbeatAck
	Resume
	Hello
	Resumed
	_
	_
	_
	ClientDisconnect
)

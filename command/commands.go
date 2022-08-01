package command

type Type int

const (
	_ Type = iota
	Heartbeat
	Identify
	UpdatePresence
	UpdateVoiceState
	_
	Resume
	_
	RequestGuildMembers
)

package opcode

const (
	EventDispatch            OpCode = 0 | internalOnly | receive
	EventHeartbeat           OpCode = 1 | internalOnly | send | receive
	EventIdentify            OpCode = 2 | internalOnly | send
	EventUpdateStatus        OpCode = 3 | send
	EventUpdateVoiceState    OpCode = 4 | send
	EventResume              OpCode = 6 | internalOnly | send
	EventReconnect           OpCode = 7 | internalOnly | receive
	EventRequestGuildMembers OpCode = 8 | send
	EventInvalidSession      OpCode = 9 | internalOnly | receive
	EventHello               OpCode = 10 | internalOnly | receive
	EventHeartbeatAck        OpCode = 11 | internalOnly | receive
)

func EventGuards(op uint8) OpCode {
	switch op {
	case EventDispatch.Val():
		return EventDispatch
	case EventHeartbeat.Val():
		return EventHeartbeat
	case EventIdentify.Val():
		return EventIdentify
	case EventUpdateStatus.Val():
		return EventUpdateStatus
	case EventUpdateVoiceState.Val():
		return EventUpdateVoiceState
	case EventResume.Val():
		return EventResume
	case EventReconnect.Val():
		return EventReconnect
	case EventRequestGuildMembers.Val():
		return EventRequestGuildMembers
	case EventInvalidSession.Val():
		return EventInvalidSession
	case EventHello.Val():
		return EventHello
	case EventHeartbeatAck.Val():
		return EventHeartbeatAck
	default:
		return Invalid
	}
}

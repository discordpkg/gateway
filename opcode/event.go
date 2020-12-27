package opcode

const (
	EventDispatch            Op = 0 | internalOnly | receive
	EventHeartbeat           Op = 1 | internalOnly | send | receive
	EventIdentify            Op = 2 | internalOnly | send
	EventUpdateStatus        Op = 3 | send
	EventUpdateVoiceState    Op = 4 | send
	EventResume              Op = 6 | internalOnly | send
	EventReconnect           Op = 7 | internalOnly | receive
	EventRequestGuildMembers Op = 8 | send
	EventInvalidSession      Op = 9 | internalOnly | receive
	EventHello               Op = 10 | internalOnly | receive
	EventHeartbeatAck        Op = 11 | internalOnly | receive
)

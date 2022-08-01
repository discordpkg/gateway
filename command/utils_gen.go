package command

func All() []Type {
	return []Type{
		Heartbeat,
		Identify,
		RequestGuildMembers,
		Resume,
		UpdatePresence,
		UpdateVoiceState,
	}
}

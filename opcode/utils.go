package opcode

func CanReconnectAfter(code Type) bool {
	_, reconnectOpCode := map[Type]bool{
		Reconnect: true,
		Resume:    true,
	}[code]

	return reconnectOpCode
}

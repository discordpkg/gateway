package closecode

func CanReconnectAfter(code Type) bool {
	_, reconnectCloseCode := map[Type]bool{
		ClientReconnecting: true,
		UnknownError:       true,
		InvalidSeq:         true,
		SessionTimedOut:    true,
	}[code]

	return reconnectCloseCode
}

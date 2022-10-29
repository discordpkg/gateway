package gatewayutil

import (
	"errors"
	"io"

	"github.com/discordpkg/gateway"
	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/opcode"
)

func HandleError(st *gateway.State, err error, closeWriter io.Writer) error {
	if errors.Is(err, gateway.ErrSequenceNumberSkipped) {
		_ = st.WriteClose(closeWriter, gateway.RestartCloseCode)
	}

	var errDiscord *gateway.DiscordError
	if errors.As(err, &errDiscord) {
		switch errDiscord.OpCode {
		case opcode.InvalidSession:
			st.InvalidateSession(closeWriter)
		case opcode.Reconnect:
			_ = st.WriteClose(closeWriter, gateway.RestartCloseCode)
		}
	}

	var websocketErr *gateway.WebsocketClosedError
	if errors.As(err, &websocketErr) {
		switch websocketErr.Code {
		case closecode.ClientReconnecting, closecode.UnknownError:
			// allow resume
		default:
			st.InvalidateSession(closeWriter)
		}
		return &gateway.DiscordError{CloseCode: closecode.Type(websocketErr.Code), Reason: websocketErr.Reason}
	}

	return err
}

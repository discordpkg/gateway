package gatewayutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/discordpkg/gateway"
	"github.com/discordpkg/gateway/event"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type WebsocketError struct {
	Err error
}

func (e *WebsocketError) Error() string {
	return fmt.Errorf("websocket logic failed: %w", e.Err).Error()
}

func (e *WebsocketError) Unwrap() error {
	return e.Err
}

type ioWriteFlusher struct {
	writer *wsutil.Writer
}

func (i *ioWriteFlusher) Write(p []byte) (n int, err error) {
	if n, err = i.writer.Write(p); err != nil {
		return n, err
	}
	return n, i.writer.Flush()
}

func NewShard(options ...gateway.Option) (*Shard, error) {
	shard := &Shard{
		options: options,
	}

	return shard, nil
}

type Shard struct {
	options []gateway.Option
	client  *gateway.Client

	Conn        net.Conn
	textWriter  io.Writer
	closeWriter io.Writer
}

type GetGatewayBotURL func() (string, error)

// Dial sets up the websocket connection before identifying with the gateway.
// The url must be complete and specify api version and encoding:
//
//	"wss://gateway.discord.gg/"                      => invalid
//	"wss://gateway.discord.gg/?v=10"                 => invalid
//	"wss://gateway.discord.gg/?v=10&encoding=json"   => valid
func (s *Shard) Dial(ctx context.Context, getURL GetGatewayBotURL) (connection net.Conn, err error) {
	dialURL := s.client.ResumeURL()
	if dialURL == "" {
		dialURL, err = getURL()
		if err != nil {
			return nil, err
		}
		if dialURL == "" {
			return nil, errors.New("unable to get a URL for websocket dial")
		}
	}

	dialURL, err = ValidateDialURL(dialURL)
	if err != nil {
		return nil, err
	}

	conn, reader, _, err := ws.Dial(ctx, dialURL)
	if err != nil {
		return nil, err
	}

	if reader != nil {
		defer ws.PutReader(reader)
		if reader.Size() > 0 {
			_ = conn.Close()
			return nil, fmt.Errorf("unable to handle data sent ... ")
			// TODO: this should be handled somehow??
		}
	}

	s.Conn = conn
	s.textWriter = s.writer(ws.OpText)
	s.closeWriter = s.writer(ws.OpClose)

	options := append(s.options, gateway.WithExistingSession(s.client))
	options = append(options, gateway.WithHeartbeatHandler(&gateway.DefaultHeartbeatHandler{
		TextWriter:       s.textWriter,
		ConnectionCloser: s.Conn,
	}))

	if s.client, err = gateway.NewClient(options...); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *Shard) Write(op event.Type, data []byte) error {
	return s.client.Write(s.textWriter, op, data)
}

func (s *Shard) writer(op ws.OpCode) io.Writer {
	return &ioWriteFlusher{wsutil.NewWriter(s.Conn, ws.StateClientSide, op)}
}

func (s *Shard) nextFrame(rd *wsutil.Reader, ctrlFrameHandler wsutil.FrameHandlerFunc) (io.Reader, error) {
	hdr, err := rd.NextFrame()
	if err != nil {
		_ = s.Conn.Close()

		msg := err.Error()
		closedConnection := strings.Contains(msg, "use of closed network connection")
		closedConnection = closedConnection || strings.Contains(msg, "use of closed connection")
		closedConnection = closedConnection || strings.Contains(msg, "i/o timeout")
		closedConnection = closedConnection || errors.Is(err, io.EOF)
		if closedConnection {
			return nil, &WebsocketError{Err: net.ErrClosed}
		} else {
			return nil, &WebsocketError{Err: err}
		}
	}
	if hdr.OpCode.IsControl() {
		// discord does send close frames so these must be handled
		if err := ctrlFrameHandler(hdr, rd); err != nil {
			var errClose wsutil.ClosedError
			if errors.As(err, &errClose) {
				mockedMessage := fmt.Sprintf("{\"closecode\":%d,\"d\":\"%s\"}", errClose.Code, errClose.Reason)
				return strings.NewReader(mockedMessage), nil
			} else {
				return nil, &WebsocketError{Err: err}
			}
		}
		return nil, nil
	}
	if hdr.OpCode&ws.OpText == 0 {
		// discord only uses text, even for heartbeats / ping/pong frames
		if err := rd.Discard(); err != nil {
			return nil, &WebsocketError{Err: err}
		}
		return nil, nil
	}

	return rd, nil
}

func (s *Shard) EventLoop() error {
	defer s.client.Close(s.closeWriter)

	controlHandler := wsutil.ControlFrameHandler(s.Conn, ws.StateClientSide)
	rd := wsutil.Reader{
		Source:          s.Conn,
		State:           ws.StateClientSide,
		CheckUTF8:       true,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}

	for {
		reader, err := s.nextFrame(&rd, controlHandler)
		if err != nil {
			return err
		} else if reader == nil {
			continue
		}

		_, err = s.client.ProcessNext(reader, s.textWriter)
		if err != nil {
			return err
		}

		return nil
	}
}

package discordgatewaygobwas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway"

	"github.com/sirupsen/logrus"
)

func writeClose(conn net.Conn, shard *discordgateway.ClientState, reason string) error {
	logrus.Info("shard sent close frame: ", reason)
	closeWriter := wsutil.NewWriter(conn, ws.StateClientSide, ws.OpClose)
	if err := shard.WriteClose(closeWriter); err != nil {
		return fmt.Errorf("failed to write close frame. %w", err)
	}
	return nil
}

// func main() {
// 	path := "wss://gateway.discord.gg/?v=8&encoding=json"
//
// 	logrus.SetLevel(logrus.DebugLevel)
// 	logrus.SetFormatter(&logrus.TextFormatter{
// 		ForceColors:               true,
// 		DisableTimestamp:          false,
// 		FullTimestamp:             true,
// 		TimestampFormat:           "",
// 	})
//
// 	conn, reader, _, err := ws.Dial(context.Background(), path)
// 	if err != nil {
// 		logrus.Fatalf("failed to open websocket connection. %w", err)
// 	}
//
// 	shard := discordgateway2.NewShard(&discordgateway2.ClientStateConfig{
// 		Token: "NDg2ODMyMjYyNTEYeP5nDE4y8c",
// 		Intents: 0b111111111111111, // everything
// 		TotalNumberOfShards: 1,
// 		Properties:discordgateway2.GatewayIdentifyProperties{
// 			OS:      "linux",
// 			Browser: "github.com/andersfylling/discordgateway v0",
// 			Device:  "tester",
// 		},
// 	})
//
// 	if reader != nil {
// 		if reader.Size() > 0 {
// 			_ = writeClose(conn, shard, "unsupported pre-events")
// 			fmt.Println("discord sent data too quickly")
// 			return
// 		}
// 		ws.PutReader(reader)
// 	}
//
// 	if err = EventLoop(conn, shard); err != nil {
// 		logrus.Error("event loop stopped: ", err)
// 	}
// }

const (
	OpCodeNone int = -1
)

func EventLoop(conn net.Conn, shard *discordgateway.ClientState) (opcode int, err error) {
	writer := wsutil.NewWriter(conn, ws.StateClientSide, ws.OpText)

	// timeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	life, kill := context.WithCancel(context.Background())
	defer func() {
		kill()
		// cancel()
	}()

	closeConnection := func() {
		if shard.Closed() {
			return
		}
		if err := writeClose(conn, shard, "program shutdown"); err != nil {
			logrus.Fatal("failed to close connection properly: ", err)
		}
		_ = conn.Close()
	}
	defer closeConnection()

	controlHandler := wsutil.ControlFrameHandler(conn, ws.StateClientSide)
	rd := wsutil.Reader{
		Source:          conn,
		State:           ws.StateClientSide,
		CheckUTF8:       true,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}
	forcedReadTimeout := atomic.Bool{}
	pulser := &heart{}
	for {
		hdr, err := rd.NextFrame()
		if err != nil {
			// check for the "historical" i/o timeout message
			// net.go@timeoutErr # ~583 at 2020-12-25
			const ioTimeoutMessage = "i/o timeout"
			if strings.Contains(err.Error(), ioTimeoutMessage) && forcedReadTimeout.Load() {
				logrus.Error("closed connection after timing out")
				_ = conn.Close()
				return OpCodeNone, nil
			} else {
				_ = conn.Close()
				return OpCodeNone, fmt.Errorf("failed to load next frame. %w", err)
			}
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				var normalClose wsutil.ClosedError
				if errors.As(err, &normalClose) {
					if forcedReadTimeout.Load() {
						_ = conn.Close()
					}
					logrus.Infof("closing down after getting %+v", normalClose)
					return OpCodeNone, &discordgateway.CloseError{Code: uint(normalClose.Code), Reason: normalClose.Reason}
				} else {
					return OpCodeNone, fmt.Errorf("failed to handle control frame. %w", err)
				}
			}
			continue
		}
		if hdr.OpCode&ws.OpText == 0 {
			if err := rd.Discard(); err != nil {
				return OpCodeNone, fmt.Errorf("failed to discard unwanted frame. %w", err)
			}
			logrus.Debugf("discarded websocket frame due to wrong op: %s", string(hdr.OpCode))
			continue
		}

		payload, length, err := shard.Read(&rd)
		if err != nil {
			logrus.Error("unable to call shard read successfully: ", err)
			break
		}

		if payload.Op > 0 {
			logrus.Debugf("read %d bytes of data, op:%d, event:%s\n", length, payload.Op, payload.EventName)
		}

		switch payload.Op {
		case 1:
			if err := shard.Heartbeat(writer); err != nil {
				return 1, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
			}
		case 7:
			logrus.Debug("discord requested a reconnect")
			return 7, nil // TODO: how to populate up that a reconnect is requested?
		case 9:
			logrus.Debug("discord invalidated session")
			return 9, nil
		case 10:
			if shard.HaveIdentified() {
				continue
			}
			if shard.HaveSessionID() {
				if err := shard.Resume(writer); err != nil {
					return 10, fmt.Errorf("sending resume failed. closing. %w", err)
				}
			} else {
				if err := shard.Identify(writer); err != nil {
					return 10, fmt.Errorf("identify failed. closing. %w", err)
				}
			}
			var hello *discordgateway.GatewayHello
			if err := json.Unmarshal(payload.Data, &hello); err != nil {
				return 10, fmt.Errorf("failed to extract heartbeat from hello message. %w", err)
			}

			pulser.gotAck.Store(true)
			pulser.interval = time.Duration(hello.HeartbeatIntervalMilli) * time.Millisecond
			pulser.conn = conn
			pulser.shard = shard
			pulser.forcedReadTimeout = &forcedReadTimeout

			go pulser.pulser(life)
		case 11:
			pulser.gotAck.Store(true)
		default:
		}
	}

	return OpCodeNone, nil
}

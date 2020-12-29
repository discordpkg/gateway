package discordgatewaygobwas

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/andersfylling/discordgateway/json"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway"
	"github.com/andersfylling/discordgateway/log"
	"github.com/andersfylling/discordgateway/opcode"
)

func writeClose(conn net.Conn, shard *discordgateway.GatewayState, reason string) error {
	log.Info("shard sent close frame: ", reason)
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

func EventLoop(conn net.Conn, shard *discordgateway.GatewayState) (opcode.OpCode, error) {
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
			log.Fatal("failed to close connection properly: ", err)
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
				log.Error("closed connection after timing out")
				_ = conn.Close()
				return opcode.Invalid, nil
			} else {
				_ = conn.Close()
				return opcode.Invalid, fmt.Errorf("failed to load next frame. %w", err)
			}
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				var normalClose wsutil.ClosedError
				if errors.As(err, &normalClose) {
					if forcedReadTimeout.Load() {
						_ = conn.Close()
					}
					log.Info(fmt.Errorf("closing down after getting %w", normalClose))
					return opcode.Invalid, &discordgateway.CloseError{Code: uint(normalClose.Code), Reason: normalClose.Reason}
				} else {
					return opcode.Invalid, fmt.Errorf("failed to handle control frame. %w", err)
				}
			}
			continue
		}
		if hdr.OpCode&ws.OpText == 0 {
			if err := rd.Discard(); err != nil {
				return opcode.Invalid, fmt.Errorf("failed to discard unwanted frame. %w", err)
			}
			log.Debug(fmt.Sprintf("discarded websocket frame due to wrong op: %s", string(hdr.OpCode)))
			continue
		}

		payload, length, err := shard.Read(&rd)
		if err != nil {
			log.Error("unable to call shard read successfully: ", err)
			break
		}

		if payload.Op != opcode.EventDispatch {
			log.Debug(fmt.Sprintf("read %d bytes of data, op:%d, event:%s\n", length, payload.Op, payload.EventName))
		}

		switch payload.Op {
		case 1:
			if err := shard.Heartbeat(writer); err != nil {
				return payload.Op, fmt.Errorf("discord requested heartbeat, but was unable to send one. %w", err)
			}
		case opcode.EventReconnect:
			return payload.Op, nil
		case 9:
			return payload.Op, nil
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
				return payload.Op, fmt.Errorf("failed to extract heartbeat from hello message. %w", err)
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

	return opcode.Invalid, nil
}

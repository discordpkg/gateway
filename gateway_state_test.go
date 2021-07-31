package discordgateway

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/closecode"
	"github.com/andersfylling/discordgateway/intent"
	"github.com/bradfitz/iter"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"

	"github.com/andersfylling/discordgateway/opcode"
)

func NewGatewayState() *GatewayState {
	return NewGatewayStateWithSeqNumber(0)
}

var defaultGatewayStateConfig = &GatewayStateConfig{
	BotToken:             "dsfsdf",
	ShardID:              0,
	TotalNumberOfShards:  1,
	Properties:           GatewayIdentifyProperties{},
	CommandRateLimitChan: nil,
	GuildEvents:          event.All(),
	DMEvents:             event.All(),
}

func NewGatewayStateWithSeqNumber(seq int64) *GatewayState {
	gs := NewGatewayClient(defaultGatewayStateConfig)
	gs.state = newStateWithSeqNumber(seq)
	return gs
}

func TestCloseError_Error(t *testing.T) {
	err := &CloseError{Code: closecode.AlreadyAuthenticated, Reason: "testing"}
	if !strings.Contains(err.Error(), strconv.Itoa(int(closecode.AlreadyAuthenticated))) {
		t.Error("missing close code")
	}
	if !strings.Contains(err.Error(), "testing") {
		t.Error("missing reason")
	}
}

func TestGatewayState_IntentGeneration(t *testing.T) {
	gs := NewGatewayState()
	if gs.intents != intent.All {
		t.Fatal("all intents should be activated")
	}
}

func TestGatewayState_Write(t *testing.T) {
	client := NewGatewayState()
	mock := &IOMock{
		writeChan: make(chan []byte, 2),
	}

	payload := []byte(`{"random":"data"}`)

	if err := client.Write(mock, opcode.RequestGuildMembers, payload); err != nil {
		t.Fatal(err)
	}

	if err := client.Write(mock, opcode.InvalidSession, payload); err == nil {
		t.Error(fmt.Errorf("should not be able to dispatch a message under a receive only op code: %w", err))
	}
}

func TestGatewayState_Read(t *testing.T) {
	client := NewGatewayState()

	t.Run("ready", func(t *testing.T) {
		t.Run("stores-session-id", func(t *testing.T) {
			sessionID := "lfhaiskge5uvrievuh"
			payloadStr := fmt.Sprintf(`{"op":0,"d":{"session_id":"%s"},"t":"%s"}`, sessionID, event.Ready)
			payload := []byte(payloadStr)

			if client.sessionID != "" {
				t.Fatal("expected sessionID to not be set")
			}

			reader := bytes.NewReader(payload)
			if _, _, err := client.Read(reader); err != nil {
				t.Fatalf("expected to be able to read message: %s", payloadStr)
			}

			if client.sessionID != sessionID {
				t.Errorf("expected session id to be '%s', but got '%s'", sessionID, client.sessionID)
			}
		})

		t.Run("require-session-id", func(t *testing.T) {
			// inject invalid json data, and expect the read to fail cause session id could not be extracted
			payloadStr := fmt.Sprintf(`{"op":0,"d":{"unknown_id":"skerugcrug"},"t":"%s"}`, event.Ready)
			payload := []byte(payloadStr)

			reader := bytes.NewReader(payload)
			if _, _, err := client.Read(reader); err == nil {
				t.Error("expected read to error on failed session id extraction")
			}
		})
	})
}

func TestGatewayState_Heartbeat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		finalSeqNumber := int64(156356)
		client := NewGatewayStateWithSeqNumber(finalSeqNumber)
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Heartbeat(mock); err != nil {
			t.Fatal("unable to send heartbeat", err)
		}

		packet, err := extractIOMockWrittenMessage(mock, 1)
		if err != nil {
			t.Fatal("message written to pipe is invalid", err)
		}

		sentSeqNumberStr := string(packet.Data)
		sentSeqNumber, err := strconv.ParseInt(sentSeqNumberStr, 10, 64)
		if err != nil {
			t.Fatal("invalid sequence number sent", err)
		}

		if sentSeqNumber != finalSeqNumber {
			t.Errorf("sequence number missmatching. Got %d, wants %d", sentSeqNumber, finalSeqNumber)
		}
	})
}
func TestGatewayState_Identify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := NewGatewayState()
		client.conf.GuildEvents = intent.Guilds.Events()
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Identify(mock); err != nil {
			t.Fatal("unable to send identify", err)
		}

		if !client.HaveIdentified() {
			t.Error("should have marked itself as identified")
		}

		packet, err := extractIOMockWrittenMessage(mock, opcode.Identify)
		if err != nil {
			t.Fatal("message written to pipe is invalid", err)
		}

		var identify *GatewayIdentify
		if err := json.Unmarshal(packet.Data, &identify); err != nil {
			t.Fatal("invalid json payload", err)
		}

		incorrect := func(name string, got, wants interface{}) {
			t.Errorf("unexpect %s. Got '%+v', wants '%+v'", name, got, wants)
		}

		if client.conf.BotToken != identify.BotToken {
			incorrect("Token", identify.BotToken, client.conf.BotToken)
		}
		if client.conf.ShardID != ShardID(identify.Shard[0]) {
			incorrect("ShardID", identify.Shard[0], client.conf.ShardID)
		}
		if client.conf.TotalNumberOfShards > 0 && client.conf.TotalNumberOfShards != identify.Shard[1] {
			incorrect("ShardCount", identify.Shard[1], client.conf.TotalNumberOfShards)
		}
		if client.intents != identify.Intents {
			incorrect("Intents", identify.Intents, client.conf.Intents)
		}
	})
	t.Run("failed-to-write", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		closedMock := &IOMockWithClosedConnection{IOMock{}}

		if err := client.Identify(closedMock); err == nil {
			t.Fatal("write should have returned a error")
		} else if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("incorrect error. Got %+v", err)
		}
	})
}
func TestGatewayState_Resume(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := NewGatewayState()
		client.conf.GuildEvents = intent.Guilds.Events()
		client.sessionID = "kdfjhsdk"
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Resume(mock); err != nil {
			t.Fatal("unable to send resume", err)
		}

		if !client.HaveIdentified() {
			t.Error("should have marked itself as identified")
		}

		packet, err := extractIOMockWrittenMessage(mock, 6)
		if err != nil {
			t.Fatal("message written to pipe is invalid", err)
		}

		var resume *GatewayResume
		if err := json.Unmarshal(packet.Data, &resume); err != nil {
			t.Fatal("invalid json payload", err)
		}

		incorrect := func(name string, v1, v2 interface{}) {
			t.Errorf("unexpect %s. Got '%+v', wants '%+v'", name, v1, v2)
		}

		if client.conf.BotToken != resume.BotToken {
			incorrect("Token", resume.BotToken, client.conf.BotToken)
		}
		if client.sessionID != resume.SessionID {
			incorrect("sessionID", resume.SessionID, client.sessionID)
		}
		if client.conf.BotToken != resume.BotToken {
			incorrect("sequence number", resume.SequenceNumber, client.SequenceNumber())
		}
	})
	t.Run("premature", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = ""
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Resume(mock); err == nil {
			t.Fatal("should not be able to resume if session id is not set")
		}
	})
	t.Run("failed-to-write", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		closedMock := &IOMockWithClosedConnection{IOMock{}}

		if err := client.Resume(closedMock); err == nil {
			t.Fatal("write should have returned a error")
		} else if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("incorrect error. Got %+v", err)
		}
	})
}
func TestGatewayState_InvalidateSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		client.InvalidateSession(mock)

		t.Run("session id", func(t *testing.T) {
			if client.sessionID != "" {
				t.Error("session id was not removed")
			}
		})

		t.Run("close code", func(t *testing.T) {
			code, err := mock.ReadCloseMessage()
			if err != nil {
				t.Fatal(err)
			}

			if code != NormalCloseCode {
				t.Errorf("incorrect close code. Got %d, wants %d", int(code), int(NormalCloseCode))
			}
		})
	})
	t.Run("failed", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		closedMock := &IOMockWithClosedConnection{IOMock{}}

		client.InvalidateSession(closedMock)
		if client.sessionID != "" {
			t.Error("session id was not removed")
		}
	})
}

func TestGatewayState_DemultiplexCloseCode(t *testing.T) {
	t.Run("should invalidate session", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.DemultiplexCloseCode(closecode.InvalidSeq, "sf", mock); err == nil {
			t.Fatal("missing error")
		}

		t.Run("session id", func(t *testing.T) {
			if client.sessionID != "" {
				t.Error("session id was not removed")
			}
		})

		t.Run("close code", func(t *testing.T) {
			//
			code, err := mock.ReadCloseMessage()
			if err != nil {
				t.Fatal(err)
			}

			if code != NormalCloseCode {
				t.Errorf("incorrect close code. Got %d, wants %d", int(code), int(NormalCloseCode))
			}
		})
	})
	t.Run("should allow session to reconnect", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.DemultiplexCloseCode(closecode.ClientReconnecting, "sf", mock); err == nil {
			t.Fatal("missing error")
		}

		t.Run("session id", func(t *testing.T) {
			if client.sessionID == "" {
				t.Error("session id was removed")
			}
		})

		t.Run("close code", func(t *testing.T) {
			code, err := mock.ReadCloseMessage()
			if err == nil {
				t.Error("there should be no close code")
			}

			if code != 0 {
				t.Errorf("got unexpected close code %d", int(code))
			}
		})
	})
}

func TestNewRateLimiter(t *testing.T) {
	t.Run("10/10ms", func(t *testing.T) {
		rl, closer := NewRateLimiter(10, 10*time.Millisecond)
		defer closer.Close()

		for range iter.N(10) {
			select {
			case <-rl:
			default:
				t.Fatal("no token available")
			}
		}

		select {
		case <-rl:
			t.Fatal("there should be no token")
		default:
		}

		<-time.After(11 * time.Millisecond)
		select {
		case <-rl:
		default:
			t.Fatal("no token available")
		}
		if len(rl) != 9 {
			t.Fatal("expected there to be only 9 tokens left")
		}
	})
}

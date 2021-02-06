package discordgateway

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/json"

	"github.com/andersfylling/discordgateway/opcode"
)

func NewGatewayState() GatewayState {
	return NewGatewayStateWithSeqNumber(0)
}

func NewGatewayStateWithSeqNumber(seq int64) GatewayState {
	return GatewayState{
		state: newStateWithSeqNumber(seq),
	}
}

func TestGatewayState_Write(t *testing.T) {
	client := NewGatewayState()
	mock := &IOMock{
		writeChan: make(chan []byte, 2),
	}

	payload := []byte(`{"random":"data"}`)

	if err := client.Write(mock, opcode.EventRequestGuildMembers, payload); err != nil {
		t.Fatal(err)
	}

	if err := client.Write(mock, opcode.EventInvalidSession, payload); err == nil {
		t.Error(fmt.Errorf("should not be able to dispatch a message under a receive only op code: %w", err))
	}
}

func TestGatewayState_Read(t *testing.T) {
	client := NewGatewayState()

	t.Run("ready", func(t *testing.T) {
		t.Run("stores-session-id", func(t *testing.T) {
			sessionID := "lfhaiskge5uvrievuh"
			payloadStr := fmt.Sprintf(`{"op":0,"d":{"session_id":"%s"},"t":"%s"}`, sessionID, event.Ready.String())
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
			payloadStr := fmt.Sprintf(`{"op":0,"d":{"unknown_id":"skerugcrug"},"t":"%s"}`, event.Ready.String())
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
		client.conf = GatewayStateConfig{
			BotToken:            "kaicyeurtbecgresn",
			Intents:             1,
			ShardID:             0,
			TotalNumberOfShards: 1,
		}
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Identify(mock); err != nil {
			t.Fatal("unable to send identify", err)
		}

		if !client.HaveIdentified() {
			t.Error("should have marked itself as identified")
		}

		packet, err := extractIOMockWrittenMessage(mock, opcode.EventIdentify)
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
		if client.conf.ShardID != identify.Shard[0] {
			incorrect("ShardID", identify.Shard[0], client.conf.ShardID)
		}
		if client.conf.TotalNumberOfShards > 0 && client.conf.TotalNumberOfShards != identify.Shard[1] {
			incorrect("ShardCount", identify.Shard[1], client.conf.TotalNumberOfShards)
		}
		if client.conf.Intents != identify.Intents {
			incorrect("Intents", identify.Intents, client.conf.Intents)
		}
	})
	t.Run("failed-to-write", func(t *testing.T) {
		client := NewGatewayState()
		client.sessionID = "sgrtxfh"
		closedMock := &IOMockWithClosedConnection{IOMock{}}

		if err := client.Identify(closedMock); err == nil {
			t.Fatal("write should have returned a error")
		} else if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("incorrect error. Got %+v", err)
		}
	})
}
func TestGatewayState_Resume(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := NewGatewayState()
		client.conf = GatewayStateConfig{
			BotToken:            "kaicyeurtbecgresn",
			Intents:             1,
			ShardID:             0,
			TotalNumberOfShards: 1,
		}
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
		} else if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("incorrect error. Got %+v", err)
		}
	})
}

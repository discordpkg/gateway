package discordgateway

import (
	"strconv"
	"testing"

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
		client.conf = ClientStateConfig{
			Token:               "kaicyeurtbecgresn",
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

		if client.conf.Token != identify.Token {
			incorrect("Token", identify.Token, client.conf.Token)
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
}
func TestGatewayState_Resume(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := NewGatewayState()
		client.conf = ClientStateConfig{
			Token:               "kaicyeurtbecgresn",
			Intents:             1,
			ShardID:             0,
			TotalNumberOfShards: 1,
		}
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

		if client.conf.Token != resume.Token {
			incorrect("Token", resume.Token, client.conf.Token)
		}
		if client.sessionID != resume.SessionID {
			incorrect("sessionID", resume.SessionID, client.sessionID)
		}
		if client.conf.Token != resume.Token {
			incorrect("sequence number", resume.SequenceNumber, client.SequenceNumber())
		}
	})
}

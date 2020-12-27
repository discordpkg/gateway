package discordgateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"
)

type IOMock struct {
	closed    bool
	writeBuf  []byte
	readBuf   io.Reader
	writeChan chan []byte
	readChan  chan []byte
}

var _ IOFlushReadWriter = &IOMock{}

func (m *IOMock) Close() error {
	m.closed = true
	return nil
}
func (m *IOMock) Flush() error {
	if len(m.writeBuf) > 0 {
		m.writeChan <- m.writeBuf
		m.writeBuf = nil
		return nil
	} else {
		return io.EOF
	}
}
func (m *IOMock) Write(p []byte) (n int, err error) {
	m.writeBuf = append(m.writeBuf, p...)
	return len(p), nil
}
func (m *IOMock) Read(p []byte) (n int, err error) {
	if m.readBuf == nil {
		select {
		case msg, ok := <-m.readChan:
			if !ok {
				return 0, io.ErrClosedPipe
			}
			m.readBuf = bytes.NewReader(msg)
		case <-time.After(time.Millisecond):
			return 0, io.EOF
		}
	}

	return m.readBuf.Read(p)
}

type IOMockWithClosedConnection struct {
	IOMock
}

func (m *IOMockWithClosedConnection) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func extractIOMockWrittenMessage(mock *IOMock, expectedOPCode uint8) (*GatewayPayload, error) {
	payload := <-mock.writeChan

	var packet *GatewayPayload
	if err := json.Unmarshal(payload, &packet); err != nil {
		return nil, fmt.Errorf("unable to unmarshal data into GatewayPayload. %w", err)
	}
	if packet.Op != expectedOPCode {
		return nil, fmt.Errorf("expected operation code %d. got %d", expectedOPCode, packet.Op)
	}
	return packet, nil
}

func TestClientState(t *testing.T) {
	t.Parallel()
	t.Run("write", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := &ClientState{}
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}

			data := []byte(`"some test data"`)
			opcode := uint8(1)
			if err := client.write(mock, opcode, data); err != nil {
				t.Fatal(err)
			}

			if mock.closed {
				t.Fatal("client was closed")
			}

			payload := <-mock.writeChan
			if len(payload) == 0 {
				t.Fatal("expected payload data")
			}

			var packet *GatewayPayload
			if err := json.Unmarshal(payload, &packet); err != nil {
				t.Error("invalid json", err)
			}

			if string(packet.Data) != string(data) {
				t.Errorf("incorrect payload data. Got %s, wants %s", packet.Data, data)
			}

			if packet.Op != opcode {
				t.Errorf("incorrect operation code. Got %d, wants %d", packet.Op, opcode)
			}
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := &ClientState{}
			mock := &IOMockWithClosedConnection{}

			err := client.write(mock, 2, nil)
			if err == nil {
				t.Fatal("expected heartbeat to fail when writing to closed connection")
			}

			if !errors.Is(err, io.ErrClosedPipe) {
				t.Fatal("close error was not io.ErrClosedPipe")
			}
		})
	})
	t.Run("read", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := &ClientState{}
			mock := &IOMock{
				readChan: make(chan []byte, 2),
			}

			heartbeatInterval := int64(4500)
			opcode := uint8(10)
			// write the data to pipe
			str := fmt.Sprintf(`{"op":%d,"d":{"heartbeat_interval":%d}}`, opcode, heartbeatInterval)
			mock.readChan <- []byte(str)

			payload, length, err := client.Read(mock)
			if err != nil {
				t.Fatal(err)
			}
			if length == 0 {
				t.Fatal("no content was read")
			}

			if mock.closed {
				t.Fatal("client was closed")
			}

			if payload.Op != opcode {
				t.Errorf("incorrect op code. Got %d, wants %d", payload.Op, opcode)
			}
			if len(payload.Data) == 0 {
				t.Fatal("payload contains no raw data")
			}

			var packet *GatewayHello
			if err := json.Unmarshal(payload.Data, &packet); err != nil {
				t.Error("invalid json", err)
			}

			if packet.HeartbeatIntervalMilli != heartbeatInterval {
				t.Errorf("incorrect interval. Got %d, wants %d", packet.HeartbeatIntervalMilli, heartbeatInterval)
			}
		})
	})
	t.Run("close", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := &ClientState{}
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}
			if err := client.WriteClose(mock); err != nil {
				t.Fatal("unable to write close code: ", err)
			}

			if !client.Closed() {
				t.Fatal("client was not closed")
			}

			payload := <-mock.writeChan
			if len(payload) == 0 {
				t.Fatal("expected payload data")
			}

			if _, _, err := client.Read(mock); !(err != nil && errors.Is(err, io.ErrClosedPipe)) {
				t.Errorf("expected closed pipe error. Got: %+v", err)
			}

			expectedCloseError := func(f func(IOFlushWriter) error) {
				if err := f(mock); !(err != nil && errors.Is(err, io.ErrClosedPipe)) {
					t.Errorf("expected closed pipe error. Got: %+v", err)
				}
			}

			expectedCloseError(client.WriteClose)
			expectedCloseError(client.Heartbeat)
			expectedCloseError(client.Identify)
			expectedCloseError(client.Resume)
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := &ClientState{}
			mock := &IOMockWithClosedConnection{}

			if err := client.WriteClose(mock); err == nil {
				t.Fatal("should fail with a 'closed pipe' error")
			}

			if !client.Closed() {
				t.Fatal("client was not closed")
			}
		})
	})
}
func TestClientState_Heartbeat(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		client := &ClientState{}
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		finalSeqNumber := int64(156356)
		client.sequenceNumber.Store(finalSeqNumber)
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
func TestClientState_Identify(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		client := &ClientState{
			conf: ClientStateConfig{
				Token:               "kaicyeurtbecgresn",
				Intents:             1,
				ShardID:             0,
				TotalNumberOfShards: 1,
			},
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

		packet, err := extractIOMockWrittenMessage(mock, 2)
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
func TestClientState_Resume(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		client := &ClientState{
			conf: ClientStateConfig{
				Token:               "kaicyeurtbecgresn",
				Intents:             1,
				ShardID:             0,
				TotalNumberOfShards: 1,
			},
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
			incorrect("sequence number", resume.SequenceNumber, client.sequenceNumber.Load())
		}
	})
}

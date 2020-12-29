package discordgateway

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/json"

	"github.com/andersfylling/discordgateway/opcode"
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

func TestClientState(t *testing.T) {
	t.Parallel()
	t.Run("write", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}

			data := []byte(`"some test data"`)
			op := opcode.EventHeartbeat
			if err := client.Write(mock, op, data); err != nil {
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

			if packet.Op.Val() != op.Val() {
				t.Errorf("incorrect operation code. Got %d, wants %d", packet.Op, op)
			}

			if packet.Op == op {
				t.Errorf("incorrect guards code. Got %d, wants %d", packet.Op, op.Val())
			}
		})
		t.Run("opcode", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}

			data := []byte(`"some test data"`)
			op := opcode.EventDispatch
			if op.Send() {
				t.Fatal("opcode must not be send-able")
			}

			t.Run("guarded", func(t *testing.T) {
				if err := client.Write(mock, op, data); err == nil {
					t.Error("should fail to write a op code which is receive only")
				}
			})
			t.Run("unguarded", func(t *testing.T) {
				op = opcode.OpCode(op.Val())
				if op.Guarded() {
					t.Fatal("guards should have been deleted")
				}
				if err := client.Write(mock, op, data); err != nil {
					t.Error("operation code should not have been guarded. ", err)
				}
			})
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := newState()
			mock := &IOMockWithClosedConnection{}

			err := client.Write(mock, opcode.EventIdentify, nil)
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
			client := newState()
			mock := &IOMock{
				readChan: make(chan []byte, 2),
			}

			heartbeatInterval := int64(4500)
			op := uint8(10)
			// write the data to pipe
			str := fmt.Sprintf(`{"op":%d,"d":{"heartbeat_interval":%d}}`, op, heartbeatInterval)
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

			if payload.Op.Val() != op {
				t.Errorf("incorrect op code. Got %d, wants %d", payload.Op, op)
			}
			if len(payload.Data) == 0 {
				t.Fatal("payload contains no raw data")
			}
		})
	})
	t.Run("close", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := newState()
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
			if err := client.WriteClose(mock); !(err != nil && errors.Is(err, io.ErrClosedPipe)) {
				t.Errorf("expected closed pipe error. Got: %+v", err)
			}
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := newState()
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

func extractIOMockWrittenMessage(mock *IOMock, expectedOPCode opcode.OpCode) (*GatewayPayload, error) {
	payload := <-mock.writeChan

	var packet *GatewayPayload
	if err := json.Unmarshal(payload, &packet); err != nil {
		return nil, fmt.Errorf("unable to unmarshal data into GatewayPayload. %w", err)
	}

	wants := opcode.OpCode(expectedOPCode.Val())
	if packet.Op != wants {
		return nil, fmt.Errorf("expected operation code %d. got %d", wants, packet.Op)
	}
	return packet, nil
}

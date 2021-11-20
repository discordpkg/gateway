package discordgateway

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/andersfylling/discordgateway/command"
	"io"
	"net"
	"testing"
	"time"

	"github.com/andersfylling/discordgateway/event"
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

var _ io.Writer = &IOMock{}
var _ io.Reader = &IOMock{}

func (m *IOMock) Close() error {
	m.closed = true
	return nil
}
func (m *IOMock) Write(p []byte) (n int, err error) {
	m.writeChan <- p
	return len(p), nil
}
func (m *IOMock) Read(p []byte) (n int, err error) {
	select {
	case msg, ok := <-m.readChan:
		if !ok {
			return 0, net.ErrClosed
		}
		m.readBuf = bytes.NewReader(msg)
	case <-time.After(time.Millisecond):
		return 0, io.EOF
	}

	return m.readBuf.Read(p)
}

func (m *IOMock) ReadCloseMessage() (uint16, error) {
	var content []byte
	select {
	case msg, ok := <-m.writeChan:
		if !ok {
			return 0, net.ErrClosed
		}
		content = msg
	case <-time.After(time.Millisecond):
		return 0, io.EOF
	}

	if len(content) != 2 {
		return 0, errors.New("incorrect close code length")
	}

	return binary.BigEndian.Uint16(content), nil
}

func (m *IOMock) CloseCode() int32 {
	code, err := m.ReadCloseMessage()
	if err != nil {
		return -1
	}
	return int32(code)
}

func (m *IOMock) NormalCloseCode() bool {
	return m.CloseCode() == int32(NormalCloseCode)
}

func (m *IOMock) RestartCloseCode() bool {
	return m.CloseCode() == int32(RestartCloseCode)
}

type IOMockWithClosedConnection struct {
	IOMock
}

func (m *IOMockWithClosedConnection) Write(p []byte) (n int, err error) {
	return 0, net.ErrClosed
}

func TestClientState(t *testing.T) {
	t.Run("write", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}

			data := []byte(`"some test data"`)
			op := command.Heartbeat
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

			if packet.Op != opcode.Type(op) {
				t.Errorf("incorrect operation code. Got %d, wants %d", packet.Op, op)
			}
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := newState()
			mock := &IOMockWithClosedConnection{}

			err := client.Write(mock, command.Identify, nil)
			if err == nil {
				t.Fatal("expected heartbeat to fail when writing to closed connection")
			}

			if !errors.Is(err, net.ErrClosed) {
				t.Fatal("close error was not net.ErrClosed")
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
			op := opcode.Hello
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

			if payload.Op != op {
				t.Errorf("incorrect op code. Got %d, wants %d", payload.Op, op)
			}
			if len(payload.Data) == 0 {
				t.Fatal("payload contains no raw data")
			}

			if client.SequenceNumber() != 0 {
				t.Errorf("expected seq to be 0 with first event")
			}

			// ensure that the sequence number increases, but skips outdated packages
			for i := 0; i < 2; i++ {
				mock.readChan <- []byte(`{"op":0,"d":{"random":"data"},"s":1}`)
				if payload, _, err = client.Read(mock); err != nil {
					t.Fatal(err)
				}

				if client.SequenceNumber() != 1 {
					t.Error("state failed to update sequence number")
				}
			}

			if !payload.Outdated {
				t.Error("when resending the same payload, it should have been marked outdated")
			}
		})
		t.Run("populates-dispatch(op:0)", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				readChan: make(chan []byte, 2),
			}

			evt := event.MessageCreate

			// write the data to pipe
			str := fmt.Sprintf(`{"op":0,"d":{"random":"data"},"t":"%s"}`, evt)
			mock.readChan <- []byte(str)

			payload, _, err := client.Read(mock)
			if err != nil {
				t.Fatal(err)
			}

			if payload.EventName != evt {
				t.Errorf("incorrect event name. Got %s, wants %s", payload.EventName, evt)
			}
		})
		t.Run("invalid-data", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				readChan: make(chan []byte, 2),
			}
			close(mock.readChan)

			_, _, err := client.Read(mock)
			if err == nil {
				t.Fatal("expected read to fail when io.Reader fails")
			}
		})
	})
	t.Run("close", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}
			if err := client.WriteNormalClose(mock); err != nil {
				t.Fatal("unable to write close code: ", err)
			}

			var data []byte
			select {
			case data = <-mock.writeChan:
			default:
				t.Fatal("nothing found on write channel")
			}

			code := binary.BigEndian.Uint16(data)
			if code != 1000 {
				t.Errorf("expected close code to be 1000, but got %d", int(code))
			}
		})
		t.Run("restart", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}
			if err := client.WriteRestartClose(mock); err != nil {
				t.Fatal("unable to write close code: ", err)
			}

			var data []byte
			select {
			case data = <-mock.writeChan:
			default:
				t.Fatal("nothing found on write channel")
			}

			code := binary.BigEndian.Uint16(data)
			if code == 1000 {
				t.Errorf("normal close code received, expected something different")
			}
		})
		t.Run("success", func(t *testing.T) {
			client := newState()
			mock := &IOMock{
				writeChan: make(chan []byte, 2),
			}
			if err := client.WriteNormalClose(mock); err != nil {
				t.Fatal("unable to write close code: ", err)
			}

			if !client.Closed() {
				t.Fatal("client was not closed")
			}

			payload := <-mock.writeChan
			if len(payload) == 0 {
				t.Fatal("expected payload data")
			}

			if _, _, err := client.Read(mock); !(err != nil && errors.Is(err, net.ErrClosed)) {
				t.Errorf("expected closed pipe error. Got: %+v", err)
			}
			if err := client.WriteNormalClose(mock); !(err != nil && errors.Is(err, net.ErrClosed)) {
				t.Errorf("expected closed pipe error. Got: %+v", err)
			}
		})
		t.Run("closed-connection", func(t *testing.T) {
			client := newState()
			mock := &IOMockWithClosedConnection{}

			shouldFail := func(err error) {
				if err == nil {
					t.Fatal("should fail fast with a 'closed pipe' error")
				}
			}

			shouldFail(client.WriteNormalClose(mock))
			shouldFail(client.Write(mock, command.Heartbeat, []byte(`{}`)))

			_, _, err := client.Read(mock)
			shouldFail(err)

			if !client.Closed() {
				t.Fatal("client was not closed")
			}
		})
	})
}

func extractIOMockWrittenMessage(mock *IOMock, expectedOPCode opcode.Type) (*GatewayPayload, error) {
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

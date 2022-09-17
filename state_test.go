package gateway

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/internal/util"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/discordpkg/gateway/closecode"
	"github.com/discordpkg/gateway/intent"

	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/json"

	"github.com/discordpkg/gateway/opcode"

	"io"
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

var defaultOptions = []Option{
	WithShardID(0),
	WithShardCount(1),
	WithIdentifyConnectionProperties(&IdentifyConnectionProperties{}),
	WithGuildEvents(event.All()...),
	WithDirectMessageEvents(event.All()...),
}

func NewDefaultState(extraOptions ...Option) *State {
	st, err := NewState("token", append(defaultOptions, extraOptions...)...)
	if err != nil {
		panic(err)
	}

	return st
}

func TestCloseError_Error(t *testing.T) {
	err := &DiscordError{CloseCode: closecode.AlreadyAuthenticated, Reason: "testing"}
	if !strings.Contains(err.Error(), strconv.Itoa(int(closecode.AlreadyAuthenticated))) {
		t.Error("missing close code")
	}
	if !strings.Contains(err.Error(), "testing") {
		t.Error("missing reason")
	}
}

func TestGatewayState_IntentGeneration(t *testing.T) {
	st := NewDefaultState()
	if st.intents != intent.Sum {
		t.Fatal("all intents should be activated")
	}
}

func TestGatewayState_Write(t *testing.T) {
	t.Run("random", func(t *testing.T) {
		client := NewDefaultState()
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		payload := []byte(`{"random":"data"}`)

		if err := client.Write(mock, command.RequestGuildMembers, payload); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("success", func(t *testing.T) {
		client := NewDefaultState()
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
		client := NewDefaultState()
		mock := &IOMockWithClosedConnection{}

		err := client.Write(mock, command.Identify, nil)
		if err == nil {
			t.Fatal("expected heartbeat to fail when writing to closed connection")
		}

		if !errors.Is(err, net.ErrClosed) {
			t.Fatal("close error was not net.ErrClosed")
		}
	})
}

func TestGatewayState_Read(t *testing.T) {

	t.Run("ready", func(t *testing.T) {
		client := NewDefaultState()
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

	t.Run("read", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := NewDefaultState()
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
			client := NewDefaultState()
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
			client := NewDefaultState()
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
}

func TestGatewayState_Close(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		client := NewDefaultState()
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
		client := NewDefaultState()
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
		client := NewDefaultState()
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
		client := NewDefaultState()
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
}

func TestGatewayState_Heartbeat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		finalSeqNumber := int64(156356)
		client := NewDefaultState(WithSequenceNumber(finalSeqNumber))
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
		client := NewDefaultState(WithGuildEvents(intent.Events(intent.Guilds)...))
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

		var identify *Identify
		if err := json.Unmarshal(packet.Data, &identify); err != nil {
			t.Fatal("invalid json payload", err)
		}

		incorrect := func(name string, got, wants interface{}) {
			t.Errorf("unexpect %s. Got '%+v', wants '%+v'", name, got, wants)
		}

		if client.botToken != identify.BotToken {
			incorrect("Token", identify.BotToken, client.botToken)
		}
		if client.shardID != ShardID(identify.Shard[0]) {
			incorrect("ShardID", identify.Shard[0], client.shardID)
		}
		if client.totalNumberOfShards > 0 && client.totalNumberOfShards != identify.Shard[1] {
			incorrect("ShardCount", identify.Shard[1], client.totalNumberOfShards)
		}
		if client.intents != identify.Intents {
			incorrect("Intents", identify.Intents, client.intents)
		}
	})
	t.Run("failed-to-write", func(t *testing.T) {
		client := NewDefaultState()
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
		client := NewDefaultState(WithGuildEvents(intent.Events(intent.Guilds)...))
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

		var resume *Resume
		if err := json.Unmarshal(packet.Data, &resume); err != nil {
			t.Fatal("invalid json payload", err)
		}

		incorrect := func(name string, v1, v2 interface{}) {
			t.Errorf("unexpect %s. Got '%+v', wants '%+v'", name, v1, v2)
		}

		if client.botToken != resume.BotToken {
			incorrect("Token", resume.BotToken, client.botToken)
		}
		if client.sessionID != resume.SessionID {
			incorrect("sessionID", resume.SessionID, client.sessionID)
		}
		if client.botToken != resume.BotToken {
			incorrect("sequence number", resume.SequenceNumber, client.SequenceNumber())
		}
	})
	t.Run("premature", func(t *testing.T) {
		client := NewDefaultState()
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.Resume(mock); err == nil {
			t.Fatal("should not be able to resume if session id is not set")
		}
	})
	t.Run("failed-to-write", func(t *testing.T) {
		client := NewDefaultState(WithSessionID("sgrtxfh"))
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
		client := NewDefaultState(WithSessionID("sgrtxfh"))
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
		client := NewDefaultState(WithSessionID("sgrtxfh"))
		closedMock := &IOMockWithClosedConnection{IOMock{}}

		client.InvalidateSession(closedMock)
		if client.sessionID != "" {
			t.Error("session id was not removed")
		}
	})
}

func TestGatewayState_DemultiplexCloseCode(t *testing.T) {
	t.Run("should invalidate session", func(t *testing.T) {
		client := NewDefaultState(WithSessionID("sgrtxfh"))
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.ProcessCloseCode(closecode.InvalidSeq, "sf", mock); err == nil {
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
		client := NewDefaultState(WithSessionID("sgrtxfh"))
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
		}

		if err := client.ProcessCloseCode(closecode.ClientReconnecting, "sf", mock); err == nil {
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

func TestGatewayState_Process(t *testing.T) {
	t.Run("should fail on sequence skipping", func(t *testing.T) {
		client := NewDefaultState(WithSessionID("sgrtxfh"))
		client.whitelist = util.Set[event.Type]{}
		client.whitelist.Add(event.MessageCreate)

		mock := &IOMock{
			writeChan: make(chan []byte, 2),
			readChan:  make(chan []byte, 2),
		}

		messageID := 2523
		payloadStr := fmt.Sprintf(`{"op":0,"d":{"id":"%d"},"t":"%s","s":%d}`, messageID, event.MessageCreate, client.SequenceNumber()+2)
		payload := []byte(payloadStr)

		_, redundant, err := client.ProcessNextMessage(bytes.NewReader(payload), mock, mock)
		if err == nil {
			t.Fatal("missing error")
		}
		if !redundant {
			t.Error("should have been redundant")
		}

		t.Run("session id", func(t *testing.T) {
			if client.sessionID == "" {
				t.Error("session id was removed")
			}
		})

		t.Run("close code", func(t *testing.T) {
			code, err := mock.ReadCloseMessage()
			if err != nil {
				t.Fatal(err)
			}

			if code != RestartCloseCode {
				t.Errorf("incorrect close code. Got %d, wants %d", int(code), int(RestartCloseCode))
			}
		})
	})
	t.Run("should fail on unknown error", func(t *testing.T) {
		client := NewDefaultState()
		client.sessionID = "sgrtxfh"
		client.whitelist = util.Set[event.Type]{}
		client.whitelist.Add(event.MessageCreate)

		mock := &IOMock{
			writeChan: make(chan []byte, 2),
			readChan:  make(chan []byte, 2),
		}

		messageID := 2523
		payloadStr := fmt.Sprintf(`{"op":0,"d":{"id":"%d"},"t":"%s","s":%d}`, messageID, event.MessageCreate, client.SequenceNumber()+2)
		payload := []byte(payloadStr + "}}}}}}") // malformed json

		_, redundant, err := client.ProcessNextMessage(bytes.NewReader(payload), mock, mock)
		if err == nil {
			t.Fatal("missing error")
		}
		if redundant {
			t.Error("unhandled errors should not be redundant")
		}
	})
	t.Run("dispatch whitelisted event", func(t *testing.T) {
		client := NewDefaultState()
		client.sessionID = "sgrtxfh"
		client.whitelist = util.Set[event.Type]{}
		client.whitelist.Add(event.MessageCreate)

		mock := &IOMock{
			writeChan: make(chan []byte, 2),
			readChan:  make(chan []byte, 2),
		}

		messageID := 2523
		payloadStr := fmt.Sprintf(`{"op":0,"d":{"id":"%d"},"t":"%s","s":%d}`, messageID, event.MessageCreate, client.SequenceNumber()+1)
		data := []byte(payloadStr)

		payload, redundant, err := client.ProcessNextMessage(bytes.NewReader(data), mock, mock)
		if err != nil {
			t.Fatal(err)
		}
		if redundant {
			t.Error("should not be redundant")
		}

		if !strings.Contains(string(payload.Data), strconv.Itoa(messageID)) {
			t.Errorf("message payload is missing message id. Got '%s'", string(payload.Data))
		}
		if client.Closed() {
			t.Error("client closed")
		}
	})
	t.Run("dispatch blacklisted event", func(t *testing.T) {
		client := NewDefaultState()
		client.sessionID = "sgrtxfh"
		client.whitelist = util.Set[event.Type]{}
		mock := &IOMock{
			writeChan: make(chan []byte, 2),
			readChan:  make(chan []byte, 2),
		}

		messageID := 2523
		payloadStr := fmt.Sprintf(`{"op":0,"d":{"id":"%d"},"t":"%s","s":%d}`, messageID, event.MessageCreate, client.SequenceNumber()+1)
		data := []byte(payloadStr)

		_, redundant, err := client.ProcessNextMessage(bytes.NewReader(data), mock, mock)
		if err != nil {
			t.Fatal("blacklisted events should not trigger an error, just a redundancy flag")
		}
		if !redundant {
			t.Error("blacklisted events are redundant")
		}
		if client.Closed() {
			t.Error("client closed")
		}
	})
}

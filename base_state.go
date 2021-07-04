package discordgateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/andersfylling/discordgateway/json"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/opcode"
)

type GatewayPayload struct {
	Op        opcode.Type     `json:"op"`
	Data      json.RawMessage `json:"d"`
	Seq       int64           `json:"s,omitempty"`
	EventName event.Type      `json:"t,omitempty"`
	Outdated  bool            `json:"-"`
}

var ErrSequenceNumberSkipped = errors.New("the sequence number increased with more than 1, events lost")

func newState() *baseState {
	return newStateWithSeqNumber(0)
}

func newStateWithSeqNumber(seq int64) *baseState {
	st := &baseState{}
	st.sequenceNumber.Store(seq)
	return st
}

type state interface {
	SequenceNumber() int64
	Closed() bool
	WriteNormalClose(client io.Writer) error
	WriteRestartClose(client io.Writer) error
	Read(client io.Reader) (*GatewayPayload, int, error)
	Write(client io.Writer, opCode opcode.Type, payload json.RawMessage) (err error)
}

// baseState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
//
// This state works for both voice and gateway.
type baseState struct {
	sequenceNumber atomic.Int64
	stateClosed    atomic.Bool
}

func (c *baseState) SequenceNumber() int64 {
	return c.sequenceNumber.Load()
}

func (c *baseState) Closed() bool {
	return c.stateClosed.Load()
}

func (c *baseState) WriteNormalClose(client io.Writer) error {
	return c.writeClose(client, 1000)
}

func (c *baseState) WriteRestartClose(client io.Writer) error {
	return c.writeClose(client, 1012)
}

func (c *baseState) writeClose(client io.Writer, code uint16) error {
	writeIfOpen := func() error {
		if c.stateClosed.CAS(false, true) {
			closeCodeBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(closeCodeBuf, code)

			_, err := client.Write(closeCodeBuf)
			return err
		}
		return net.ErrClosed
	}

	if err := writeIfOpen(); err != nil {
		if !errors.Is(err, net.ErrClosed) && strings.Contains(err.Error(), "use of closed connection") {
			return net.ErrClosed
		}
		return err
	}
	return nil
}

// Read until a new data frame with updated info comes in, or fails.
func (c *baseState) Read(client io.Reader) (*GatewayPayload, int, error) {
	if c.stateClosed.Load() {
		return nil, 0, net.ErrClosed
	}

	data, err := ioutil.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &GatewayPayload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	prevSeq := c.sequenceNumber.Load()
	packet.Outdated = prevSeq >= packet.Seq
	if packet.Seq-prevSeq > 1 {
		return nil, 0, ErrSequenceNumberSkipped
	}
	if !packet.Outdated {
		c.sequenceNumber.Store(packet.Seq)
	}

	return packet, len(data), nil
}

func (c *baseState) Write(client io.Writer, opCode opcode.Type, payload json.RawMessage) (err error) {
	if c.stateClosed.Load() {
		return net.ErrClosed
	}

	defer func() {
		if err != nil {
			// TODO: skip error wrapping if the connection was closed before hand
			//  no need to close twice and pretend this was the first place to
			//  do so..
			// _ = client.Close()
			err = fmt.Errorf("client after failed dispatch. %w", err)
		}
	}()

	packet := GatewayPayload{
		Op:   opCode,
		Data: payload,
	}

	var data []byte
	data, err = json.Marshal(&packet)
	if err != nil {
		return fmt.Errorf("unable to marshal packet; %w", err)
	}

	_, err = client.Write(data)
	return err
}

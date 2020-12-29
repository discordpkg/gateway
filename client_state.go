package discordgateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/andersfylling/discordgateway/json"

	"go.uber.org/atomic"

	"github.com/andersfylling/discordgateway/event"
	"github.com/andersfylling/discordgateway/log"
	"github.com/andersfylling/discordgateway/opcode"
)

type GatewayPayload struct {
	Op        opcode.OpCode   `json:"op"`
	Data      json.RawMessage `json:"d"`
	Seq       int64           `json:"s,omitempty"`
	EventName string          `json:"t,omitempty"`
	EventFlag event.Flag      `json:"-"`
	Outdated  bool            `json:"-"`
}

func newState() *clientState {
	return newStateWithSeqNumber(0)
}

func newStateWithSeqNumber(seq int64) *clientState {
	st := &clientState{}
	st.sequenceNumber.Store(seq)
	return st
}

type state interface {
	SequenceNumber() int64
	Closed() bool
	WriteClose(client IOFlushWriter) error
	Read(client IOReader) (*GatewayPayload, int, error)
	Write(client IOFlushWriter, opCode opcode.OpCode, payload json.RawMessage) (err error)
}

// clientState should be discarded after the connection has closed.
// reconnect must create a new shard instance.
type clientState struct {
	sequenceNumber atomic.Int64
	stateClosed    atomic.Bool
}

func (c *clientState) SequenceNumber() int64 {
	return c.sequenceNumber.Load()
}

func (c *clientState) Closed() bool {
	return c.stateClosed.Load()
}

func (c *clientState) WriteClose(client IOFlushWriter) error {
	if c.stateClosed.CAS(false, true) {
		// there is no real benefit to give Discord a reason.
		// relevant errors should instead be logged for diagnostic purposes.
		closeCodeBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(closeCodeBuf, uint16(1000))

		if _, err := client.Write(closeCodeBuf); err != nil {
			return fmt.Errorf("unable to write close code to Discord. %w", err)
		}
		return client.Flush()
	}
	return io.ErrClosedPipe
}

// Read until a new data frame with updated info comes in, or fails.
func (c *clientState) Read(client IOReader) (*GatewayPayload, int, error) {
	if c.stateClosed.Load() {
		return nil, 0, io.ErrClosedPipe
	}

	data, err := ioutil.ReadAll(client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read data. %w", err)
	}

	packet := &GatewayPayload{}
	if err = json.Unmarshal(data, packet); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal packet. %w", err)
	}

	// add opcode guards
	// however, discord might send new opcode values
	guardedOpCode := opcode.EventGuards(packet.Op.Val())
	if guardedOpCode != opcode.Invalid {
		packet.Op = guardedOpCode
	}

	// set event flags
	if packet.Op == opcode.EventDispatch {
		if packet.EventFlag, err = event.StringToEvent(packet.EventName); err != nil {
			log.Error(fmt.Sprintf("event flag for event name %s does not exist", packet.EventName))
		}
	}

	prevSeq := c.sequenceNumber.Load()
	packet.Outdated = prevSeq >= packet.Seq
	if !packet.Outdated {
		c.sequenceNumber.Store(packet.Seq)
	}
	if packet.Seq-prevSeq > 1 {
		log.Debug("sequence number jumped by ", packet.Seq-prevSeq)
	}

	return packet, len(data), nil
}

func (c *clientState) Write(client IOFlushWriter, opCode opcode.OpCode, payload json.RawMessage) (err error) {
	if opCode.Guarded() && !opCode.Send() {
		return fmt.Errorf("the guarded operation code is not send-able. Code %d", int(opCode.Val()))
	}
	if c.stateClosed.Load() {
		return io.ErrClosedPipe
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

	var length int
	length, err = client.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send packet; %w", err)
	}
	if length == 0 {
		return errors.New("no data was sent")
	}

	return client.Flush()
}

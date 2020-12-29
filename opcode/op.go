package opcode

import (
	"strconv"
)

const (
	Invalid OpCode = valueMask | internalOnly | invalid
)

// Op is the operation bitmask
//
// first 5 bits are reserved (highest valued bits),
// while the remaining hold the actual op code.
type OpCode uint16

func (op OpCode) MarshalJSON() (data []byte, err error) {
	strVal := op.String()
	return []byte(strVal), nil
}

func (op OpCode) Val() uint8 {
	return uint8(valueMask & op)
}

func (op OpCode) Send() bool {
	return op.Guarded() && (op&send) > 0
}

func (op OpCode) Receive() bool {
	return op.Guarded() && (op&receive) > 0
}

func (op OpCode) Voice() bool {
	return op.Guarded() && (op&voice) > 0
}

func (op OpCode) InternalUseOnly() bool {
	return op.Guarded() && (op&internalOnly) > 0
}

func (op OpCode) Guarded() bool {
	return (op & reservedMask) > 0
}

func (op OpCode) String() string {
	v := int(op.Val())
	return strconv.Itoa(v)
}

const (
	size                    = 16
	reservedMaskSize OpCode = 5
	reservedMask     OpCode = 0b11111 << (size - reservedMaskSize)
	highestBit       OpCode = 1 << (size - 1)
	valueMask        OpCode = (^OpCode(0)) ^ reservedMask
)

const (
	send         OpCode = reservedMask & (highestBit)
	receive      OpCode = reservedMask & (highestBit >> 1)
	internalOnly OpCode = reservedMask & (highestBit >> 2)
	voice        OpCode = reservedMask & (highestBit >> 3)
	invalid      OpCode = reservedMask & (highestBit >> 4)
)

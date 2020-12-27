package opcode

// Op is the operation bitmask
//
// first 4 bits are reserved (highest valued bits),
// while the remaining hold the actual op code.
type Op uint16

func (op Op) Val() uint8 {
	return uint8(valueMask & op)
}

func (op Op) Send() bool {
	return (op & send) > 0
}

func (op Op) Receive() bool {
	return (op & receive) > 0
}

func (op Op) InternalUseOnly() bool {
	return (op & internalOnly) > 0
}

// String get string representation of the op code
// Op(8) => REQUEST_GUILD_MEMBERS
func (op Op) String() string {
	panic("not implemented")
}

const (
	size            = 16
	highestBit   Op = 1 << (size - 1)
	reservedMask Op = 0b1111 << (size - 4)
	valueMask    Op = (^Op(0)) ^ reservedMask
)

const (
	send         Op = reservedMask & (highestBit)
	receive      Op = reservedMask & (highestBit >> 1)
	internalOnly Op = reservedMask & (highestBit >> 2)
)

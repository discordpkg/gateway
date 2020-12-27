package opcode

import "testing"

func TestConstants(t *testing.T) {
	if highestBit == 0 {
		t.Error("highestBit is not set")
	}
	if reservedMask == 0 {
		t.Error("reservedMask is not set")
	}
	if reservedMask>>(size-4) != 0b1111 {
		t.Error("reservedMask is incorrectly set")
	}
	if valueMask == 0 {
		t.Error("valueMask is not set")
	}
	if send == 0 {
		t.Error("send is not set")
	}
	if receive == 0 {
		t.Error("receive is not set")
	}
	if internalOnly == 0 {
		t.Error("internalOnly is not set")
	}
}

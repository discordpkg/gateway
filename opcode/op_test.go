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

func TestEventGuards(t *testing.T) {
	for i := OpCode(0); i < 20; i++ {
		v := EventGuards(i.Val())
		if v == Invalid {
			continue
		}

		if v.Voice() {
			t.Errorf("opcode should not be voice related. Code %d", int(i))
		}

		if !v.InternalUseOnly() {
			if !v.Send() {
				t.Errorf("if opcode is not limited to internal use, it must be a send-able. Code %d", int(i))
			}
		}

		if (v.Receive() || v.Send()) == false {
			t.Errorf("opcode does not have directional guard defined. Code %d", int(i))
		}
	}
}

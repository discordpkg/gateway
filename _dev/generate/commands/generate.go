package commands

import "strconv"

type Info struct {
	Name        string
	Description string
	Opcode      int
}

func (c Info) String() string {
	if c.Opcode < 0 {
		panic("invalid command value / opcode")
	}

	return strconv.FormatUint(uint64(c.Opcode), 10)
}

package event

import (
	"bytes"
	"testing"
)

func getSize() int {
	return 1750035 + 0
}

func grow1(b *bytes.Buffer) {
	max := getSize() / 512
	for i := 1; i <= max; i++ {
		b.Grow(512 * i)
	}
}

func grow2(b *bytes.Buffer) {
	b.Grow(getSize())
}

func BenchmarkFib1(t *testing.B) {
	t.ReportAllocs()
	var by []byte
	for n := 0; n < t.N; n++ {
		var b bytes.Buffer
		grow1(&b)
		by = b.Bytes()
		by = append(by, '4')
	}
}

func BenchmarkFib2(t *testing.B) {
	t.ReportAllocs()
	var by []byte
	for n := 0; n < t.N; n++ {
		var b bytes.Buffer
		grow2(&b)
		by = b.Bytes()
		by = append(by, '4')
	}
}

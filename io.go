package discordgateway

import (
	"io"
)

type IOFlusher interface {
	Flush() error
}

type IOWriter interface {
	io.Writer
}

type IOReader interface {
	io.Reader
}

type IOReadWriter interface {
	IOWriter
	IOReader
}

type IOFlushWriter interface {
	IOFlusher
	IOWriter
}

type IOFlushReadWriter interface {
	IOFlusher
	IOReader
	IOWriter
}

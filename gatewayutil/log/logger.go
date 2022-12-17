package log

import "github.com/discordpkg/gateway"

var LogInstance gateway.Logger = &nop{}

func Debug(format string, args ...interface{}) {
	LogInstance.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	LogInstance.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	LogInstance.Warn(format, args)
}

func Error(format string, args ...interface{}) {
	LogInstance.Error(format, args...)
}

func Panic(format string, args ...interface{}) {
	LogInstance.Panic(format, args...)
}

type nop struct{}

func (n *nop) Debug(_ string, _ ...interface{}) {}
func (n *nop) Info(_ string, _ ...interface{})  {}
func (n *nop) Warn(_ string, _ ...interface{})  {}
func (n *nop) Error(_ string, _ ...interface{}) {}
func (n *nop) Panic(_ string, _ ...interface{}) {}

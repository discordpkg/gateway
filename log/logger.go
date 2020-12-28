package log

// Logger for logging different situations
type Logger interface {
	// Debug low level insight in system behavior to assist diagnostic.
	Debug(args ...interface{})

	// Info general information that might be interesting
	Info(args ...interface{})

	// Warn creeping technical debt, such as dependency updates will cause the system to not compile/break.
	Warn(args ...interface{})

	// Error recoverable events/issues that does not cause a system shutdown, but is also crucial and needs to be
	// dealt with quickly.
	Error(args ...interface{})

	// Fatal identifies system crashing/breaking issues that forces the application to shut down or completely stop
	Fatal(args ...interface{})
}

var LogInstance Logger = &nop{}

func Debug(args ...interface{}) {
	LogInstance.Debug(args...)
}
func Info(args ...interface{}) {
	LogInstance.Info(args...)
}
func Warn(args ...interface{}) {
	LogInstance.Warn(args)
}
func Error(args ...interface{}) {
	LogInstance.Error(args...)
}
func Fatal(args ...interface{}) {
	LogInstance.Fatal(args...)
}

type nop struct{}

func (n *nop) Debug(_ ...interface{}) {}
func (n *nop) Info(_ ...interface{})  {}
func (n *nop) Warn(_ ...interface{})  {}
func (n *nop) Error(_ ...interface{}) {}
func (n *nop) Fatal(_ ...interface{}) {}

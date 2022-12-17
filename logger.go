package gateway

// Logger for logging different situations
type Logger interface {
	// Debug low level insight in system behavior to assist diagnostic.
	Debug(format string, args ...interface{})

	// Info general information that might be interesting
	Info(format string, args ...interface{})

	// Warn creeping technical debt, such as dependency updates will cause the system to not compile/break.
	Warn(format string, args ...interface{})

	// Error recoverable events/issues that does not cause a system shutdown, but is also crucial and needs to be
	// dealt with quickly.
	Error(format string, args ...interface{})

	// Panic identifies system crashing/breaking issues that forces the application to shut down or completely stop
	Panic(format string, args ...interface{})
}

type nopLogger struct{}

func (n *nopLogger) Debug(_ string, _ ...interface{}) {}
func (n *nopLogger) Info(_ string, _ ...interface{})  {}
func (n *nopLogger) Warn(_ string, _ ...interface{})  {}
func (n *nopLogger) Error(_ string, _ ...interface{}) {}
func (n *nopLogger) Panic(_ string, _ ...interface{}) {}

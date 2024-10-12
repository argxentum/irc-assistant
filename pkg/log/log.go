package log

type Severity int

const (
	Default   Severity = 0
	Debug     Severity = 100 // Debug or trace information
	Info      Severity = 200 // Routine information, such as ongoing status or performance
	Notice    Severity = 300 // Normal but significant events, such as start up, shut down, or a configuration change
	Warning   Severity = 400 // Warning events might cause problems
	Error     Severity = 500 // Error events are likely to cause problems
	Critical  Severity = 600 // Critical events cause more severe problems or outages
	Alert     Severity = 700 // A person must take an action immediately
	Emergency Severity = 800 // One or more systems are unusable
)

type Labeler interface {
	Labels() map[string]string
}

type Log interface {
	Close() error
	Log(l Labeler, message string, severity Severity)
	Default(l Labeler, message any)
	Defaultf(l Labeler, format string, args ...any)
	Debug(l Labeler, message any)
	Debugf(l Labeler, format string, args ...any)
	Info(l Labeler, message any)
	Infof(l Labeler, format string, args ...any)
	Notice(l Labeler, message any)
	Noticef(l Labeler, format string, args ...any)
	Warning(l Labeler, message any)
	Warningf(l Labeler, format string, args ...any)
	Error(l Labeler, message any)
	Errorf(l Labeler, format string, args ...any)
	Critical(l Labeler, message any)
	Criticalf(l Labeler, format string, args ...any)
	Alert(l Labeler, message any)
	Alertf(l Labeler, format string, args ...any)
	Emergency(l Labeler, message any)
	Emergencyf(l Labeler, format string, args ...any)
	Rawf(severity Severity, format string, args ...any)
}

var logger Log = nil

func Logger() Log {
	if logger == nil {
		panic("logger is not initialized")
	}

	return logger
}

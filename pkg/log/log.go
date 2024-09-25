package log

import (
	"assistant/pkg/api/irc"
)

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

type Log interface {
	Close() error
	Log(e *irc.Event, message string, severity Severity)
	Default(e *irc.Event, message any)
	Defaultf(e *irc.Event, format string, args ...any)
	Debug(e *irc.Event, message any)
	Debugf(e *irc.Event, format string, args ...any)
	Info(e *irc.Event, message any)
	Infof(e *irc.Event, format string, args ...any)
	Notice(e *irc.Event, message any)
	Noticef(e *irc.Event, format string, args ...any)
	Warning(e *irc.Event, message any)
	Warningf(e *irc.Event, format string, args ...any)
	Error(e *irc.Event, message any)
	Errorf(e *irc.Event, format string, args ...any)
	Critical(e *irc.Event, message any)
	Criticalf(e *irc.Event, format string, args ...any)
	Alert(e *irc.Event, message any)
	Alertf(e *irc.Event, format string, args ...any)
	Emergency(e *irc.Event, message any)
	Emergencyf(e *irc.Event, format string, args ...any)
}

var logger Log = nil

func Logger() Log {
	if logger == nil {
		panic("logger is not initialized")
	}

	return logger
}

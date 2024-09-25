package log

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
	"cloud.google.com/go/logging"
	"fmt"
	"google.golang.org/api/option"
	"strings"
	"time"
)

var logger Log = nil

type Log interface {
	Close() error
	Log(e *irc.Event, message string, severity logging.Severity)
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

func Logger() Log {
	if logger == nil {
		panic("logger is not initialized")
	}

	return logger
}

func Initialize(ctx context.Context, config *config.Config) (Log, error) {
	if logger != nil {
		return logger, nil
	}

	client, err := logging.NewClient(ctx, config.GoogleCloud.ProjectID, option.WithCredentialsFile(config.GoogleCloud.ServiceAccountFilename))

	logger = &gcpLogger{
		ctx:    ctx,
		client: client,
		logger: client.Logger(config.Connection.Nick),
	}

	return logger, err
}

type gcpLogger struct {
	ctx    context.Context
	client *logging.Client
	logger *logging.Logger
}

func (l *gcpLogger) Close() error {
	return l.client.Close()
}

func (l *gcpLogger) Log(e *irc.Event, message string, severity logging.Severity) {
	l.logger.Log(logging.Entry{Payload: message, Severity: severity, Labels: createLabels(e)})
}

func (l *gcpLogger) Default(e *irc.Event, message any) {
	l.Defaultf(e, "%s", message)
}

func (l *gcpLogger) Defaultf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Default)
	fmt.Printf("%s [-] %s\n", timestamp(), message)
}

func (l *gcpLogger) Debug(e *irc.Event, message any) {
	l.Debugf(e, "%s", message)
}

func (l *gcpLogger) Debugf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Debug)
	fmt.Printf("%s [D] %s\n", timestamp(), message)
}

func (l *gcpLogger) Info(e *irc.Event, message any) {
	l.Infof(e, "%s", message)
}

func (l *gcpLogger) Infof(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Info)
	fmt.Printf("%s [I] %s\n", timestamp(), message)
}

func (l *gcpLogger) Notice(e *irc.Event, message any) {
	l.Noticef(e, "%s", message)
}

func (l *gcpLogger) Noticef(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Notice)
	fmt.Printf("%s [N] %s\n", timestamp(), message)
}

func (l *gcpLogger) Warning(e *irc.Event, message any) {
	l.Warningf(e, "%s", message)
}

func (l *gcpLogger) Warningf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Warning)
	fmt.Printf("%s [W] %s\n", timestamp(), message)
}

func (l *gcpLogger) Error(e *irc.Event, message any) {
	l.Errorf(e, "%s", message)
}

func (l *gcpLogger) Errorf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Error)
	fmt.Printf("%s [E] %s\n", timestamp(), message)
}

func (l *gcpLogger) Critical(e *irc.Event, message any) {
	l.Criticalf(e, "%s", message)
}

func (l *gcpLogger) Criticalf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Critical)
	fmt.Printf("%s [X] %s\n", timestamp(), message)
}

func (l *gcpLogger) Alert(e *irc.Event, message any) {
	l.Alertf(e, "%s", message)
}

func (l *gcpLogger) Alertf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Alert)
	fmt.Printf("%s [Y] %s\n", timestamp(), message)
}

func (l *gcpLogger) Emergency(e *irc.Event, message any) {
	l.Emergencyf(e, "%s", message)
}

func (l *gcpLogger) Emergencyf(e *irc.Event, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(e, message, logging.Emergency)
	fmt.Printf("%s [Z] %s\n", timestamp(), message)
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

func createLabels(e *irc.Event) map[string]string {
	labels := make(map[string]string)
	labels["id"] = e.ID
	labels["code"] = e.Code
	labels["raw"] = e.Raw
	labels["from"] = e.From
	labels["source"] = e.Source
	labels["arguments"] = fmt.Sprintf("[%s]", strings.Join(e.Arguments, ", "))
	labels["is_private_message"] = fmt.Sprintf("%t", e.IsPrivateMessage())

	from, fromType := e.Sender()
	to, toType := e.Recipient()

	if e.Code == irc.CodePrivateMessage && len(from) > 0 {
		labels["entity_from"] = fmt.Sprintf("%s::%s", fromType, from)
		labels["entity_to"] = fmt.Sprintf("%s::%s", toType, to)
	} else if len(from) > 0 && len(e.Source) > 0 {
		labels["entity_from"] = fmt.Sprintf("%s::%s (%s)", fromType, from, e.Source)
	} else {
		labels["entity_from"] = fmt.Sprintf("%s", e.From)
	}

	return labels
}

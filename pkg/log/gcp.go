package log

import (
	"assistant/pkg/config"
	"cloud.google.com/go/logging"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"time"
)

func InitializeGCPLogger(ctx context.Context, cfg *config.Config, logID string) (Log, error) {
	if logger != nil {
		return logger, nil
	}

	client, err := logging.NewClient(ctx, cfg.GoogleCloud.ProjectID, option.WithCredentialsFile(cfg.GoogleCloud.ServiceAccountFilename))

	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, fmt.Errorf("error creating logging client")
	}

	logger = &gcpLogger{
		ctx:    ctx,
		client: client,
		logger: client.Logger(logID),
	}

	return logger, err
}

type gcpLogger struct {
	ctx    context.Context
	client *logging.Client
	logger *logging.Logger
}

func (gl *gcpLogger) Close() error {
	return gl.client.Close()
}

func (gl *gcpLogger) Log(l Labeler, message string, severity Severity) {
	var labels map[string]string
	if l != nil {
		labels = l.Labels()
	}
	gl.logger.Log(logging.Entry{Payload: message, Severity: logging.Severity(severity), Labels: labels})
}

func (gl *gcpLogger) Rawf(severity Severity, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.logger.Log(logging.Entry{Payload: message, Severity: logging.Severity(severity)})
	fmt.Printf("%s [ ] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Default(l Labeler, message any) {
	gl.Defaultf(l, "%s", message)
}

func (gl *gcpLogger) Defaultf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Default)
	fmt.Printf("%s [-] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Debug(l Labeler, message any) {
	gl.Debugf(l, "%s", message)
}

func (gl *gcpLogger) Debugf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Debug)
	fmt.Printf("%s [D] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Info(l Labeler, message any) {
	gl.Infof(l, "%s", message)
}

func (gl *gcpLogger) Infof(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Info)
	fmt.Printf("%s [I] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Notice(l Labeler, message any) {
	gl.Noticef(l, "%s", message)
}

func (gl *gcpLogger) Noticef(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Notice)
	fmt.Printf("%s [N] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Warning(l Labeler, message any) {
	gl.Warningf(l, "%s", message)
}

func (gl *gcpLogger) Warningf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Warning)
	fmt.Printf("%s [W] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Error(l Labeler, message any) {
	gl.Errorf(l, "%s", message)
}

func (gl *gcpLogger) Errorf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Error)
	fmt.Printf("%s [E] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Critical(l Labeler, message any) {
	gl.Criticalf(l, "%s", message)
}

func (gl *gcpLogger) Criticalf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Critical)
	fmt.Printf("%s [X] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Alert(l Labeler, message any) {
	gl.Alertf(l, "%s", message)
}

func (gl *gcpLogger) Alertf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Alert)
	fmt.Printf("%s [Y] %s\n", timestamp(), message)
}

func (gl *gcpLogger) Emergency(l Labeler, message any) {
	gl.Emergencyf(l, "%s", message)
}

func (gl *gcpLogger) Emergencyf(l Labeler, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	gl.Log(l, message, Emergency)
	fmt.Printf("%s [Z] %s\n", timestamp(), message)
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}

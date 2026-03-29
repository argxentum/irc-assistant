package modes

import (
	"assistant/pkg/api/irc"
	"time"
)

// ChannelMode represents an exclusive operating mode for a channel.
// When active, normal command processing is bypassed and all messages
// are routed to the mode's HandleEvent method.
type ChannelMode interface {
	Name() string
	Channel() string
	HandleEvent(e *irc.Event)
	OnStart()
	OnEnd()
	Timeout() time.Duration
	AllowCommand(commandName string) bool
}

package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
	"time"
)

const uptimeFunctionName = "uptime"

type uptimeFunction struct {
	stub
}

func NewUptimeFunction(ctx context.Context, cfg *config.Config, irc core.IRC) (Function, error) {
	stub, err := newFunctionStub(ctx, cfg, irc, uptimeFunctionName)
	if err != nil {
		return nil, err
	}

	return &uptimeFunction{
		stub: stub,
	}, nil
}

func (f *uptimeFunction) Matches(e *core.Event) bool {
	if !f.isAuthorized(e) {
		return false
	}

	tokens := sanitizedTokens(e.Message(), 200)

	for _, p := range f.Prefixes {
		if tokens[0] == p {
			return true
		}
	}
	return false
}

func (f *uptimeFunction) Execute(e *core.Event) error {
	startedAt := f.ctx.Value(context.StartedAtKey).(time.Time)
	elapsed := time.Since(startedAt)
	years := int(elapsed.Hours() / 24 / 365)
	elapsed -= time.Duration(years) * 24 * 365 * time.Hour
	months := int(elapsed.Hours() / 24 / 30)
	elapsed -= time.Duration(months) * 24 * 30 * time.Hour
	days := int(elapsed.Hours() / 24)
	elapsed -= time.Duration(days) * 24 * time.Hour
	hours := int(elapsed.Hours())
	elapsed -= time.Duration(hours) * time.Hour
	minutes := int(elapsed.Minutes())
	elapsed -= time.Duration(minutes) * time.Minute
	seconds := int(elapsed.Seconds())

	response := "My uptime: "
	if years > 0 {
		plural := ""
		if years > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d year%s, ", years, plural)
	}
	if months > 0 || years > 0 {
		plural := ""
		if months > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d month%s, ", months, plural)
	}
	if days > 0 || months > 0 || years > 0 {
		plural := ""
		if days > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d day%s, ", days, plural)
	}
	if hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if hours > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d hour%s, ", hours, plural)
	}
	if minutes > 0 || hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if minutes > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d minute%s, ", minutes, plural)
	}
	if seconds > 0 || minutes > 0 || hours > 0 || days > 0 || months > 0 || years > 0 {
		plural := ""
		if seconds > 1 {
			plural = "s"
		}
		response += fmt.Sprintf("%d second%s", seconds, plural)
	}

	f.irc.SendMessage(e.ReplyTarget(), response)
	return nil
}

package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

func Route(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (Function, error) {
	switch name {
	case echoFunctionName:
		return NewEchoFunction(ctx, cfg, irc)
	case sayFunctionName:
		return NewSayFunction(ctx, cfg, irc)
	case helpFunctionName:
		return NewHelpFunction(ctx, cfg, irc)
	case joinFunctionName:
		return NewJoinFunction(ctx, cfg, irc)
	case leaveFunctionName:
		return NewLeaveFunction(ctx, cfg, irc)
	case uptimeFunctionName:
		return NewUptimeFunction(ctx, cfg, irc)
	case kickFunctionName:
		return NewKickFunction(ctx, cfg, irc)
	case banFunctionName:
		return NewBanFunction(ctx, cfg, irc)
	case sleepFunctionName:
		return NewSleepFunction(ctx, cfg, irc)
	case wakeFunctionName:
		return NewWakeFunction(ctx, cfg, irc)
	case aboutFunctionName:
		return NewAboutFunction(ctx, cfg, irc)
	case searchFunctionName:
		return NewSearchFunction(ctx, cfg, irc)
	case "r/politics":
		return NewRedditFunction("politics", ctx, cfg, irc)
	case "r/news":
		return NewRedditFunction("news", ctx, cfg, irc)
	case "r/worldnews":
		return NewRedditFunction("worldnews", ctx, cfg, irc)
	case summaryFunctionName:
		return NewSummaryFunction(ctx, cfg, irc)
	case biasFunctionName:
		return NewBiasFunction(ctx, cfg, irc)
	case "bing/simple/time":
		return NewBingSimpleAnswerFunction(
			"time",
			"current date and time in %s",
			"%s: %s on %s",
			"",
			1,
			ctx, cfg, irc,
		)
	case "bing/simple/election":
		return NewBingSimpleAnswerFunction(
			"election",
			"when is the next election day",
			"%s is %s %s",
			"Note: early voting and state/local election dates differ by location. More info: https://www.usa.gov/when-to-vote",
			0,
			ctx, cfg, irc,
		)
	case tempBanFunctionName:
		//return NewTempBanFunction(ctx, cfg, irc)
	}

	return nil, fmt.Errorf("unknown function: %s", name)
}

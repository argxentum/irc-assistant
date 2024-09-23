package functions

import (
	"assistant/config"
	"assistant/pkg/api/context"
	"assistant/pkg/api/core"
	"fmt"
)

var loadedFunctions = make(map[string]Function)

func Route(ctx context.Context, cfg *config.Config, irc core.IRC, name string) (Function, error) {
	switch name {
	case echoFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewEchoFunction(ctx, cfg, irc)
		})
	case sayFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewSayFunction(ctx, cfg, irc)
		})
	case helpFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewHelpFunction(ctx, cfg, irc)
		})
	case joinFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewJoinFunction(ctx, cfg, irc)
		})
	case leaveFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewLeaveFunction(ctx, cfg, irc)
		})
	case uptimeFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewUptimeFunction(ctx, cfg, irc)
		})
	case kickFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewKickFunction(ctx, cfg, irc)
		})
	case banFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewBanFunction(ctx, cfg, irc)
		})
	case sleepFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewSleepFunction(ctx, cfg, irc)
		})
	case wakeFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewWakeFunction(ctx, cfg, irc)
		})
	case aboutFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewAboutFunction(ctx, cfg, irc)
		})
	case pollsFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewPollsFunction(ctx, cfg, irc)
		})
	case searchFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewSearchFunction(ctx, cfg, irc)
		})
	case "r/politics":
		return loadFunction(name, func() (Function, error) {
			return NewRedditFunction("politics", ctx, cfg, irc)
		})
	case "r/news":
		return loadFunction(name, func() (Function, error) {
			return NewRedditFunction("news", ctx, cfg, irc)
		})
	case "r/worldnews":
		return loadFunction(name, func() (Function, error) {
			return NewRedditFunction("worldnews", ctx, cfg, irc)
		})
	case "r/UkrainianConflict":
		return loadFunction(name, func() (Function, error) {
			return NewRedditFunction("UkrainianConflict", ctx, cfg, irc)
		})
	case summaryFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewSummaryFunction(ctx, cfg, irc)
		})
	case biasFunctionName:
		return loadFunction(name, func() (Function, error) {
			return NewBiasFunction(ctx, cfg, irc)
		})
	case "bing/simple/time":
		return loadFunction(name, func() (Function, error) {
			return NewBingSimpleAnswerFunction(
				"time",
				"current date and time in %s",
				"%s: %s on %s",
				"",
				1,
				ctx, cfg, irc,
			)
		})
	case "bing/simple/election":
		return loadFunction(name, func() (Function, error) {
			return NewBingSimpleAnswerFunction(
				"election",
				"when is the next election day",
				"%s is %s %s",
				"Note: early voting and state/local election dates differ by location. More info: https://www.usa.gov/when-to-vote",
				0,
				ctx, cfg, irc,
			)
		})
	}

	return nil, fmt.Errorf("unknown function: %s", name)
}

func loadFunction(name string, creation func() (Function, error)) (Function, error) {
	if f, ok := loadedFunctions[name]; ok {
		return f, nil
	}

	var err error
	loadedFunctions[name], err = creation()
	return loadedFunctions[name], err
}

package functions

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
)

var registry FunctionRegistry

type FunctionRegistry interface {
	Function(name string) Function
	Functions() map[string]Function
	FunctionsSortedForProcessing() []Function
	RegisterFunctions()
}

type functionRegistry struct {
	ctx       context.Context
	cfg       *config.Config
	irc       irc.IRC
	functions map[string]Function
}

func LoadFunctionRegistry(ctx context.Context, cfg *config.Config, irc irc.IRC) FunctionRegistry {
	if registry != nil {
		return registry
	}

	registry = &functionRegistry{
		ctx:       ctx,
		cfg:       cfg,
		irc:       irc,
		functions: make(map[string]Function),
	}

	registry.RegisterFunctions()
	return registry
}

func (fr *functionRegistry) Function(name string) Function {
	if f, ok := fr.functions[name]; ok {
		return f
	}

	return nil
}

func (fr *functionRegistry) Functions() map[string]Function {
	return fr.functions
}

func (fr *functionRegistry) FunctionsSortedForProcessing() []Function {
	triggered := make([]Function, 0)
	nonTriggered := make([]Function, 0)
	for _, f := range fr.functions {
		if len(f.Triggers()) == 0 {
			nonTriggered = append(nonTriggered, f)
		} else {
			triggered = append(triggered, f)
		}
	}
	triggered = append(triggered, nonTriggered...)
	return triggered
}

func (fr *functionRegistry) RegisterFunctions() {
	fr.functions[helpFunctionName] = NewHelpFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[uptimeFunctionName] = NewUptimeFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[aboutFunctionName] = NewAboutFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[pollsFunctionName] = NewPollsFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[predictItFunctionName] = NewPredictItFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[searchFunctionName] = NewSearchFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[summaryFunctionName] = NewSummaryFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[biasFunctionName] = NewBiasFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[marketsFunctionName] = NewMarketsFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[stockFunctionName] = NewStockFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[currencyFunctionName] = NewCurrencyFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[karmaSetFunctionName] = NewKarmaSetFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[karmaGetFunctionName] = NewKarmaGetFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[reminderFunctionName] = NewReminderFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[animatedTextFunctionName] = NewAnimatedTextFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[gifSearchFunctionName] = NewGifSearchFunction(fr.ctx, fr.cfg, fr.irc)

	fr.functions["r/politics"] = NewRedditFunction(
		fr.ctx, fr.cfg, fr.irc,
		"politics",
		"Searches for a recent r/politics post on the given topic.",
		[]string{"politics"},
		[]string{"%s <topic>"},
	)

	fr.functions["r/news"] = NewRedditFunction(
		fr.ctx, fr.cfg, fr.irc,
		"news",
		"Searches for a recent r/news post on the given topic.",
		[]string{"news"},
		[]string{"%s <topic>"},
	)

	fr.functions["r/worldnews"] = NewRedditFunction(
		fr.ctx, fr.cfg, fr.irc,
		"worldnews",
		"Searches for a recent r/worldnews post on the given topic.",
		[]string{"worldnews"},
		[]string{"%s <topic>"},
	)

	fr.functions["r/UkrainianConflict"] = NewRedditFunction(
		fr.ctx, fr.cfg, fr.irc,
		"UkrainianConflict",
		"Searches for a recent r/UkrainianConflict post on the given topic.",
		[]string{"ukraine"},
		[]string{"%s <topic>"},
	)

	fr.functions["bing/simple/time"] = NewBingSimpleAnswerFunction(
		fr.ctx, fr.cfg, fr.irc,
		[]string{"time"},
		[]string{"%s <location>"},
		"Displays the date and time of the given location.",
		"time", "current date and time in %s",
		"%s: %s on %s",
		"",
		1,
	)

	fr.functions["bing/simple/election"] = NewBingSimpleAnswerFunction(
		fr.ctx, fr.cfg, fr.irc,
		[]string{"election"},
		[]string{"%s"},
		"Displays the next election date.",
		"election",
		"when is the next election day",
		"%s is %s %s",
		"Note: early voting and state/local election dates differ by location. More info: https://www.usa.gov/when-to-vote",
		0,
	)

	fr.functions[echoFunctionName] = NewEchoFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[sayFunctionName] = NewSayFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[joinFunctionName] = NewJoinFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[leaveFunctionName] = NewLeaveFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[kickFunctionName] = NewKickFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[banFunctionName] = NewBanFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[kickBanFunctionName] = NewKickBanFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[tempBanFunctionName] = NewTempBanFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[bannedWordAddFunctionName] = NewBannedWordAddFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[bannedWordDeleteFunctionName] = NewBannedWordDeleteFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[sleepFunctionName] = NewSleepFunction(fr.ctx, fr.cfg, fr.irc)
	fr.functions[wakeFunctionName] = NewWakeFunction(fr.ctx, fr.cfg, fr.irc)
}

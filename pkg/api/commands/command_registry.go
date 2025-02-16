package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/config"
)

var registry CommandRegistry

type CommandRegistry interface {
	Command(name string) Command
	Commands() map[string]Command
	CommandsSortedForProcessing() []Command
	RegisterCommands()
}

type commandRegistry struct {
	ctx      context.Context
	cfg      *config.Config
	irc      irc.IRC
	commands map[string]Command
}

func LoadCommandRegistry(ctx context.Context, cfg *config.Config, irc irc.IRC) CommandRegistry {
	if registry != nil {
		return registry
	}

	registry = &commandRegistry{
		ctx:      ctx,
		cfg:      cfg,
		irc:      irc,
		commands: make(map[string]Command),
	}

	registry.RegisterCommands()
	return registry
}

func (cr *commandRegistry) Command(name string) Command {
	if f, ok := cr.commands[name]; ok {
		return f
	}

	return nil
}

func (cr *commandRegistry) Commands() map[string]Command {
	return cr.commands
}

var orderedCommands = make([]Command, 0)

func (cr *commandRegistry) CommandsSortedForProcessing() []Command {
	if len(orderedCommands) > 0 {
		return orderedCommands
	}

	nonTriggered := make([]Command, 0)
	for _, f := range cr.commands {
		if len(f.Triggers()) == 0 {
			nonTriggered = append(nonTriggered, f)
		} else {
			orderedCommands = append(orderedCommands, f)
		}
	}
	orderedCommands = append(orderedCommands, nonTriggered...)
	return orderedCommands
}

func (cr *commandRegistry) RegisterCommands() {
	cr.commands[HelpCommandName] = NewHelpCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[UptimeCommandName] = NewUptimeCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[AboutCommandName] = NewAboutCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[PollsCommandName] = NewPollsCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[PredictItCommandName] = NewPredictItCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SearchCommandName] = NewSearchCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SummaryCommandName] = NewSummaryCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SourceCommandName] = NewSourceCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[MarketsCommandName] = NewMarketsCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[StockCommandName] = NewStockCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[CurrencyCommandName] = NewCurrencyCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[KarmaSetCommandName] = NewKarmaSetCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[KarmaGetCommandName] = NewKarmaGetCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[ReminderCommandName] = NewReminderCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[RemindersCommandName] = NewRemindersCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[AnimatedTextCommandName] = NewAnimatedTextCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[GIFSearchCommandName] = NewGifSearchCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[WikipediaCommandName] = NewWikipediaCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SeenCommandName] = NewSeenCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[QuoteAddCommandName] = NewQuoteAddCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[QuotesSearchCommandName] = NewQuotesSearchCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[QuoteRandomCommandName] = NewQuoteRandomCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[NoteAddCommandName] = NewNoteAddCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[NoteDeleteCommandName] = NewNoteDeleteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[NotesSearchCommandName] = NewNotesSearchCommand(cr.ctx, cr.cfg, cr.irc)

	cr.commands["r/politics"] = NewRedditCommand(
		cr.ctx, cr.cfg, cr.irc,
		"politics",
		"Searches for a recent r/politics post on the given topic.",
		[]string{"politics"},
		[]string{"%s <topic>"},
	)

	cr.commands["r/news"] = NewRedditCommand(
		cr.ctx, cr.cfg, cr.irc,
		"news",
		"Searches for a recent r/news post on the given topic.",
		[]string{"news"},
		[]string{"%s <topic>"},
	)

	cr.commands["r/worldnews"] = NewRedditCommand(
		cr.ctx, cr.cfg, cr.irc,
		"worldnews",
		"Searches for a recent r/worldnews post on the given topic.",
		[]string{"worldnews"},
		[]string{"%s <topic>"},
	)

	cr.commands["r/UkrainianConflict"] = NewRedditCommand(
		cr.ctx, cr.cfg, cr.irc,
		"UkrainianConflict",
		"Searches for a recent r/UkrainianConflict post on the given topic.",
		[]string{"ukraine"},
		[]string{"%s <topic>"},
	)

	cr.commands["bing/simple/time"] = NewBingSimpleAnswerCommand(
		cr.ctx, cr.cfg, cr.irc,
		[]string{"time"},
		[]string{"%s <location>"},
		"Displays the date and time of the given location.",
		"time", "current date and time in %s",
		"%s: %s on %s",
		"",
		1,
	)

	cr.commands["bing/simple/election"] = NewBingSimpleAnswerCommand(
		cr.ctx, cr.cfg, cr.irc,
		[]string{"election"},
		[]string{"%s"},
		"Displays the next election date.",
		"election",
		"when is the next US election day",
		"%s is %s",
		"More info: https://www.usa.gov/voting-and-elections",
		0,
	)

	// commands requiring authorization
	cr.commands[SourceAddCommandName] = NewSourceAddCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[EchoCommandName] = NewEchoCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SayCommandName] = NewSayCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[JoinCommandName] = NewJoinCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[LeaveCommandName] = NewLeaveCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[MuteCommandName] = NewMuteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[UnmuteCommandName] = NewUnmuteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[TempMuteCommandName] = NewTempMuteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[AutoVoiceCommandName] = NewAutoVoiceCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[VoiceRequestCommandName] = NewVoiceRequestCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[VoiceRequestManagementCommandName] = NewVoiceRequestManagementCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[KickCommandName] = NewKickCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[BanCommandName] = NewBanCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[UnbanCommandName] = NewUnbanCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[KickBanCommandName] = NewKickBanCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[TempBanCommandName] = NewTempBanCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[BannedWordAddCommandName] = NewBannedWordAddCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[BannedWordDeleteCommandName] = NewBannedWordDeleteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[DisinfoWarningAddCommandName] = NewDisinfoWarningAddCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[DisinfoWarningDeleteCommandName] = NewDisinfoWarningDeleteCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[SleepCommandName] = NewSleepCommand(cr.ctx, cr.cfg, cr.irc)
	cr.commands[WakeCommandName] = NewWakeCommand(cr.ctx, cr.cfg, cr.irc)
}

package modes

import (
	"assistant/pkg/api/elapse"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/api/trivia"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	triviaModeName = "trivia"

	triviaStateAnnouncing = iota
	triviaStateAskingQuestion
	triviaStateWaitingForAnswer
	triviaStatePausing
	triviaStateShowingResults
	triviaStateEnded
)

const (
	defaultQuestionTimeout = 30 * time.Second
	defaultGameTimeout     = 10 * time.Minute
	announcementDelay      = 3 * time.Second
	pauseBetweenQuestions  = 3 * time.Second
)

type TriviaMode struct {
	channel         string
	ircs            irc.IRC
	cfg             *config.Config
	questions       []trivia.Question
	currentIndex    int
	scores          map[string]int
	state           int
	answered        bool
	responded       map[string]bool
	firstAnswerOnly bool
	answeredCh      chan struct{}
	mu              sync.Mutex
	cancel          chan struct{}
	ended           bool
}

func NewTriviaMode(channel string, ircs irc.IRC, cfg *config.Config, questions []trivia.Question) *TriviaMode {
	return &TriviaMode{
		channel:         channel,
		ircs:            ircs,
		cfg:             cfg,
		questions:       questions,
		scores:          make(map[string]int),
		responded:       make(map[string]bool),
		firstAnswerOnly: true,
		state:           triviaStateAnnouncing,
		cancel:          make(chan struct{}),
	}
}

func (t *TriviaMode) SetFirstAnswerOnly(v bool) { t.firstAnswerOnly = v }

func (t *TriviaMode) firstAnswerHint() string {
	if t.firstAnswerOnly {
		return style.Bold("but only your first answer is accepted")
	}
	return style.Bold("you can answer as many times as you like")
}

func (t *TriviaMode) Name() string    { return triviaModeName }
func (t *TriviaMode) Channel() string { return t.channel }

func (t *TriviaMode) Timeout() time.Duration {
	if t.cfg.Trivia.GameTimeout != "" {
		if d, err := time.ParseDuration(t.cfg.Trivia.GameTimeout); err == nil {
			return d
		}
	}
	return defaultGameTimeout
}

func (t *TriviaMode) AllowCommand(commandName string) bool {
	switch commandName {
	case "trivia", "sleep", "wake":
		return true
	}
	return false
}

func (t *TriviaMode) questionTimeout() time.Duration {
	if t.cfg.Trivia.QuestionTimeout != "" {
		if d, err := time.ParseDuration(t.cfg.Trivia.QuestionTimeout); err == nil {
			return d
		}
	}
	return defaultQuestionTimeout
}

func (t *TriviaMode) OnStart() {
	logger := log.Logger()
	logger.Infof(nil, "trivia mode starting in %s (%d questions)", t.channel, len(t.questions))

	q := t.questions[0]
	t.ircs.SendMessages(t.channel, []string{
		fmt.Sprintf("🎯 Trivia time! %d questions (%s / %s). Anyone can play! Respond with the answer number, %s. Please note: normal commands are paused until the game ends.", len(t.questions), q.Category, q.Difficulty, t.firstAnswerHint()),
	})

	select {
	case <-time.After(announcementDelay):
		t.askQuestion()
	case <-t.cancel:
		return
	}
}

func (t *TriviaMode) OnEnd() {
	t.mu.Lock()
	if t.ended {
		t.mu.Unlock()
		return
	}
	t.ended = true
	close(t.cancel)
	showResults := t.state != triviaStateShowingResults && t.state != triviaStateEnded
	t.state = triviaStateEnded
	t.mu.Unlock()

	if showResults && len(t.scores) > 0 {
		t.sendResults()
	}

	cooldown := t.cooldownDuration()
	msg := "🎯 Trivia has ended! Normal commands are now active."
	if cooldown > 0 {
		msg += fmt.Sprintf(" Next trivia available %s.", elapse.FutureTimeDescription(time.Now().Add(cooldown)))
	}

	// send the ending message after results with a short delay
	time.Sleep(1 * time.Second)
	t.ircs.SendMessages(t.channel, []string{msg})
}

func (t *TriviaMode) cooldownDuration() time.Duration {
	if t.cfg.Trivia.Cooldown != "" {
		if d, err := time.ParseDuration(t.cfg.Trivia.Cooldown); err == nil {
			return d
		}
	}
	return 0
}

func (t *TriviaMode) HandleEvent(e *irc.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state != triviaStateWaitingForAnswer || t.answered {
		return
	}

	msg := strings.TrimSpace(e.Message())
	if len(msg) == 0 {
		return
	}

	// treat any message starting with a digit as an answer attempt
	numStr := strings.Fields(msg)[0]
	num, err := strconv.Atoi(numStr)
	isAnswer := err == nil && num >= 1 && num <= len(t.questions[t.currentIndex].Answers)

	if t.firstAnswerOnly && t.responded[e.From] {
		return
	}
	if isAnswer {
		t.responded[e.From] = true
	} else {
		return
	}

	q := t.questions[t.currentIndex]
	if num == q.CorrectIndex {
		t.answered = true
		t.scores[e.From]++
		t.ircs.SendMessages(t.channel, []string{
			fmt.Sprintf("✅ %s got it! The answer is [%d] %s", e.From, q.CorrectIndex, q.Answers[q.CorrectIndex-1]),
		})
		select {
		case t.answeredCh <- struct{}{}:
		default:
		}
	}
}

func (t *TriviaMode) askQuestion() {
	t.mu.Lock()
	if t.ended {
		t.mu.Unlock()
		return
	}

	q := t.questions[t.currentIndex]
	t.state = triviaStateAskingQuestion
	t.answered = false
	t.responded = make(map[string]bool)
	t.answeredCh = make(chan struct{}, 1)
	t.mu.Unlock()

	lines := []string{
		fmt.Sprintf("❓ Question %d/%d: %s", t.currentIndex+1, len(t.questions), q.Question),
	}

	for i, a := range q.Answers {
		lines = append(lines, fmt.Sprintf("  [%d] %s", i+1, a))
	}

	t.ircs.SendMessages(t.channel, lines)

	t.mu.Lock()
	t.state = triviaStateWaitingForAnswer
	t.mu.Unlock()

	timeout := t.questionTimeout()
	select {
	case <-time.After(timeout):
		t.mu.Lock()
		if !t.answered && !t.ended {
			t.mu.Unlock()
			t.ircs.SendMessages(t.channel, []string{
				fmt.Sprintf("⏰ Time's up! The answer was [%d] %s", q.CorrectIndex, q.Answers[q.CorrectIndex-1]),
			})
		} else {
			t.mu.Unlock()
		}
	case <-t.answeredCh:
		// correct answer received, advance immediately
	case <-t.cancel:
		return
	}

	t.advanceToNext()
}

func (t *TriviaMode) advanceToNext() {
	t.mu.Lock()
	if t.ended {
		t.mu.Unlock()
		return
	}
	t.state = triviaStatePausing
	t.currentIndex++
	hasMore := t.currentIndex < len(t.questions)
	t.mu.Unlock()

	if !hasMore {
		t.showResultsAndEnd()
		return
	}

	select {
	case <-time.After(pauseBetweenQuestions):
		t.askQuestion()
	case <-t.cancel:
		return
	}
}

func (t *TriviaMode) showResultsAndEnd() {
	t.mu.Lock()
	t.state = triviaStateShowingResults
	t.mu.Unlock()

	t.sendResults()
	time.Sleep(1 * time.Second)

	GetManager().Deactivate(t.channel)
}

func (t *TriviaMode) sendResults() {
	if len(t.scores) == 0 {
		t.ircs.SendMessages(t.channel, []string{"🎯 No one scored any points!"})
		return
	}

	type entry struct {
		nick  string
		score int
	}

	entries := make([]entry, 0, len(t.scores))
	for nick, score := range t.scores {
		entries = append(entries, entry{nick, score})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	medals := []string{"🥇", "🥈", "🥉"}
	lines := []string{"🏆 Final Results:"}
	for i, e := range entries {
		if i >= 3 {
			break
		}
		medal := medals[i]
		pts := "pts"
		if e.score == 1 {
			pts = "pt"
		}
		lines = append(lines, fmt.Sprintf("%s %s — %d %s", medal, e.nick, e.score, pts))
	}

	t.ircs.SendMessages(t.channel, lines)
}

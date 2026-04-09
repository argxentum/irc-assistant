package main

import (
	"assistant/pkg/api/wikipedia"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultSessionTimeout = 10 * time.Minute
const streamTimeout = 30 * time.Second
const streamContentThreshold = 300
const maxHistoryAssistantLength = 200
const maxHistoryMessages = 40

var thinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)
var searchPattern = regexp.MustCompile(`\[SEARCH:\s*(.+?)\]`)

type session struct {
	id         string
	messages   []ollamaMessage
	lastActive time.Time
}

func (p *proxy) sessionTimeout() time.Duration {
	if p.cfg.Proxy.Ollama.SessionTimeout != "" {
		if d, err := time.ParseDuration(p.cfg.Proxy.Ollama.SessionTimeout); err == nil {
			return d
		}
	}
	return defaultSessionTimeout
}

func (p *proxy) getOrCreateSession(channel string) (*session, string, []ollamaMessage) {
	timeout := p.sessionTimeout()

	p.mu.Lock()
	defer p.mu.Unlock()

	s, ok := p.sessions[channel]
	if !ok || time.Since(s.lastActive) > timeout {
		s = &session{id: uuid.NewString()}
		p.sessions[channel] = s
	}
	s.lastActive = time.Now()
	return s, s.id, append([]ollamaMessage{}, s.messages...)
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaStreamChunk struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

// streamResult holds the outcome of streaming an Ollama request.
type streamResult struct {
	content   strings.Builder
	contentMu sync.Mutex
	done      chan struct{}
	complete  bool
}

func (sr *streamResult) snapshot() string {
	sr.contentMu.Lock()
	defer sr.contentMu.Unlock()
	return strings.TrimSpace(thinkPattern.ReplaceAllString(sr.content.String(), ""))
}

// streamOllamaChat sends a streaming request to Ollama and collects chunks.
// It returns immediately after the stream completes, the content threshold is reached, or the timeout fires.
// The caller can wait on result.done to know when streaming finishes fully.
func (p *proxy) streamOllamaChat(messages []ollamaMessage, options map[string]any, start time.Time) (*streamResult, error) {
	logger := log.Logger()

	req := ollamaRequest{
		Model:    p.cfg.Proxy.Ollama.Model,
		Messages: messages,
		Stream:   true,
		Options:  options,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling ollama request: %w", err)
	}

	httpResp, err := http.Post(p.cfg.Proxy.Ollama.Endpoint+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error calling ollama: %w", err)
	}
	logger.Debugf(nil, "[timing] http.Post returned in %s", time.Since(start))

	if httpResp.StatusCode != http.StatusOK {
		httpResp.Body.Close()
		return nil, fmt.Errorf("ollama returned status %d", httpResp.StatusCode)
	}

	sr := &streamResult{done: make(chan struct{})}
	thresholdReady := make(chan struct{})
	var thresholdOnce sync.Once

	go func() {
		defer httpResp.Body.Close()
		defer close(sr.done)
		firstChunk := true
		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			if firstChunk {
				logger.Debugf(nil, "[timing] first chunk received at %s", time.Since(start))
				firstChunk = false
			}
			var chunk ollamaStreamChunk
			if json.Unmarshal(line, &chunk) != nil {
				continue
			}

			sr.contentMu.Lock()
			sr.content.WriteString(chunk.Message.Content)
			strippedLen := len(strings.TrimSpace(thinkPattern.ReplaceAllString(sr.content.String(), "")))
			sr.contentMu.Unlock()

			if strippedLen >= streamContentThreshold {
				thresholdOnce.Do(func() {
					logger.Debugf(nil, "[timing] content threshold (%d chars) reached at %s", streamContentThreshold, time.Since(start))
					close(thresholdReady)
				})
			}
			if chunk.Done {
				return
			}
		}
	}()

	// Wait for stream completion, content threshold, or timeout
	timer := time.NewTimer(streamTimeout)
	select {
	case <-sr.done:
		timer.Stop()
		sr.complete = true
		logger.Debugf(nil, "[timing] stream completed in %s", time.Since(start))
	case <-thresholdReady:
		timer.Stop()
		logger.Debugf(nil, "[timing] content threshold reached, responding early at %s", time.Since(start))
	case <-timer.C:
		logger.Debugf(nil, "[timing] stream timed out after %s", time.Since(start))
	}

	return sr, nil
}

var funSearchDescriptions = []string{
	"searching the archives",
	"flipping through pages",
	"digging through stacks of paper",
	"checking my sources",
	"doing some research",
	"hitting the books",
	"looking that up",
	"asking the librarian",
	"querying the mainframe",
	"downloading more RAM",
	"phoning a friend in #wikipedia",
	"alt-tabbing to Wikipedia",
}

func (p *proxy) handleLLM(requestID string, data models.ProxyLLMRequestTaskData) error {
	logger := log.Logger()
	logger.Debugf(nil, "LLM request from %s in %s: %s", data.Nick, data.Channel, data.Prompt)

	s, sessionID, history := p.getOrCreateSession(data.Channel)

	messages := []ollamaMessage{}
	if p.cfg.Proxy.Ollama.Prompt != "" {
		prompt := strings.NewReplacer(
			"{nick}", p.cfg.IRC.Nick,
			"{channel}", data.Channel,
			"{server}", p.cfg.IRC.ServerName,
		).Replace(p.cfg.Proxy.Ollama.Prompt)
		messages = append(messages, ollamaMessage{Role: "system", Content: prompt})
	}
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}
	for _, msg := range history {
		if msg.Role == "assistant" && len(msg.Content) > maxHistoryAssistantLength {
			msg.Content = truncateAtSentence(msg.Content, maxHistoryAssistantLength)
		}
		messages = append(messages, msg)
	}
	userContent := data.Nick + ": " + data.Prompt
	messages = append(messages, ollamaMessage{Role: "user", Content: userContent})

	logger.Debugf(nil, "ollama request: %d messages", len(messages))
	for i, msg := range messages {
		preview := msg.Content
		if len(preview) > 25 {
			preview = preview[:25] + "..."
		}
		logger.Debugf(nil, "  [%d] %s: %s", i, msg.Role, preview)
	}

	options := map[string]any{"num_predict": 1024, "temperature": 0.2, "num_ctx": 8192}
	start := time.Now()

	// Streaming request
	sr, err := p.streamOllamaChat(messages, options, start)
	if err != nil {
		return err
	}

	snapshot := sr.snapshot()

	// If stream completed and model requested a search, execute it and re-prompt
	if sr.complete {
		if match := searchPattern.FindStringSubmatch(snapshot); len(match) == 2 {
			query := strings.TrimSpace(match[1])
			logger.Debugf(nil, "model requested search: %q", query)

			// Notify the user
			desc := funSearchDescriptions[rand.Intn(len(funSearchDescriptions))]
			notifyTask := models.NewProxySummaryResponseTask(data.Channel, data.Nick, "", "", []string{
				fmt.Sprintf("%s: one moment, %s...", data.Nick, desc),
			})
			if err := queue.GetDefault().Publish(notifyTask); err != nil {
				logger.Warningf(nil, "error publishing search notification: %s", err)
			}

			// Execute the search
			searchResult := executeWikipediaSearch(query, p.cfg.Reddit.UserAgent)
			logger.Debugf(nil, "[timing] wikipedia search for %q completed in %s", query, time.Since(start))

			// Append the assistant's search request and the result, then re-prompt
			messages = append(messages, ollamaMessage{Role: "assistant", Content: snapshot})
			messages = append(messages, ollamaMessage{Role: "user", Content: fmt.Sprintf("[System: search results for %q]\n\n%s", query, searchResult)})

			logger.Debugf(nil, "search follow-up request: %d messages", len(messages))
			for i, msg := range messages {
				preview := msg.Content
				if len(preview) > 50 {
					preview = preview[:50] + "..."
				}
				logger.Debugf(nil, "  [%d] %s: %s", i, msg.Role, preview)
			}

			sr, err = p.streamOllamaChat(messages, options, start)
			if err != nil {
				return err
			}
			snapshot = sr.snapshot()
		}
	}

	// Strip any leftover [SEARCH: ...] tags from the final response
	snapshot = searchPattern.ReplaceAllString(snapshot, "")
	snapshot = strings.TrimSpace(snapshot)

	// Log raw content for debugging empty responses
	sr.contentMu.Lock()
	rawContent := sr.content.String()
	sr.contentMu.Unlock()
	logger.Debugf(nil, "[timing] snapshot length: %d, complete: %v, raw length: %d", len(snapshot), sr.complete, len(rawContent))
	if len(snapshot) == 0 && len(rawContent) > 0 {
		preview := rawContent
		if len(preview) > 200 {
			preview = preview[:200]
		}
		logger.Debugf(nil, "raw content stripped to empty: %q", preview)
	}

	if len(snapshot) == 0 {
		return fmt.Errorf("empty response from ollama")
	}

	// Update session history immediately for complete responses
	if sr.complete {
		p.mu.Lock()
		s.messages = append(s.messages, ollamaMessage{Role: "user", Content: userContent})
		s.messages = append(s.messages, ollamaMessage{Role: "assistant", Content: snapshot})
		s.lastActive = time.Now()
		p.mu.Unlock()
	}

	fs := firestore.Get()
	r := models.NewLLMResponse(requestID, sessionID, data.Channel, data.Nick, p.cfg.Proxy.Ollama.Model, data.Prompt, snapshot, sr.complete)
	if err = fs.CreateLLMResponse(r); err != nil {
		return fmt.Errorf("error saving LLM response to firestore: %w", err)
	}

	logger.Debugf(nil, "LLM response saved to firestore for %s in %s [complete: %v]", data.Nick, data.Channel, sr.complete)

	if err = p.publishResponse(requestID, data.Channel, data.Nick, r.ID, sessionID, !sr.complete); err != nil {
		return err
	}

	// If still streaming, update Firestore and session history when done
	if !sr.complete {
		go func() {
			<-sr.done
			final := sr.snapshot()
			final = searchPattern.ReplaceAllString(final, "")
			final = strings.TrimSpace(final)

			p.mu.Lock()
			s.messages = append(s.messages, ollamaMessage{Role: "user", Content: userContent})
			s.messages = append(s.messages, ollamaMessage{Role: "assistant", Content: final})
			s.lastActive = time.Now()
			p.mu.Unlock()

			if err := fs.UpdateLLMResponse(r.ID, final); err != nil {
				logger.Errorf(nil, "error updating LLM response %s: %s", r.ID, err)
			} else {
				logger.Debugf(nil, "LLM response %s completed and updated in firestore", r.ID)
			}
		}()
	}

	return nil
}

func executeWikipediaSearch(query, userAgent string) string {
	logger := log.Logger()
	logger.Debugf(nil, "executing wikipedia search for: %s", query)

	page, err := wikipedia.Search(query, userAgent)
	if err != nil {
		logger.Warningf(nil, "wikipedia search error: %s", err)
		return fmt.Sprintf("Wikipedia search failed: %s", err)
	}

	if page == nil {
		return "No Wikipedia results found for this query."
	}

	result := fmt.Sprintf("Title: %s\n\n%s", page.Title, page.Extract)
	if page.ContentURLs.Desktop.Page != "" {
		result += fmt.Sprintf("\n\nSource: %s", page.ContentURLs.Desktop.Page)
	}

	return result
}

func truncateAtSentence(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// search backwards from maxLen for a sentence-ending punctuation
	for i := maxLen; i > 0; i-- {
		if s[i-1] == '.' || s[i-1] == '!' || s[i-1] == '?' {
			return s[:i]
		}
	}
	// no sentence boundary found, hard cut
	return s[:maxLen]
}

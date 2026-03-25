package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultSessionTimeout = 10 * time.Minute
const streamTimeout = 30 * time.Second
const streamContentThreshold = 200
const maxHistoryAssistantLength = 200

var thinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

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

	req := ollamaRequest{
		Model:    p.cfg.Proxy.Ollama.Model,
		Messages: messages,
		Stream:   true,
		Options:  map[string]any{"num_predict": 1024, "temperature": 0.2, "num_ctx": 8192},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling ollama request: %w", err)
	}

	start := time.Now()
	httpResp, err := http.Post(p.cfg.Proxy.Ollama.Endpoint+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error calling ollama: %w", err)
	}
	logger.Debugf(nil, "[timing] http.Post returned in %s", time.Since(start))

	if httpResp.StatusCode != http.StatusOK {
		httpResp.Body.Close()
		return fmt.Errorf("ollama returned status %d", httpResp.StatusCode)
	}

	// Stream tokens into content buffer until done, threshold reached, or timeout
	var (
		content        strings.Builder
		contentMu      sync.Mutex
		streamDone     = make(chan struct{})
		thresholdReady = make(chan struct{})
		thresholdOnce  sync.Once
	)

	go func() {
		defer httpResp.Body.Close()
		defer close(streamDone)
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
			contentMu.Lock()
			content.WriteString(chunk.Message.Content)
			strippedLen := len(strings.TrimSpace(thinkPattern.ReplaceAllString(content.String(), "")))
			contentMu.Unlock()
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
	complete := false
	timer := time.NewTimer(streamTimeout)
	select {
	case <-streamDone:
		timer.Stop()
		complete = true
		logger.Debugf(nil, "[timing] stream completed in %s", time.Since(start))
	case <-thresholdReady:
		timer.Stop()
		logger.Debugf(nil, "[timing] content threshold reached, responding early at %s", time.Since(start))
	case <-timer.C:
		logger.Debugf(nil, "[timing] stream timed out after %s", time.Since(start))
	}

	contentMu.Lock()
	snapshot := strings.TrimSpace(thinkPattern.ReplaceAllString(content.String(), ""))
	contentMu.Unlock()
	logger.Debugf(nil, "[timing] snapshot length: %d, complete: %v", len(snapshot), complete)

	if len(snapshot) == 0 {
		return fmt.Errorf("empty response from ollama")
	}

	// Update session history immediately for complete responses
	if complete {
		p.mu.Lock()
		s.messages = append(s.messages, ollamaMessage{Role: "user", Content: userContent})
		s.messages = append(s.messages, ollamaMessage{Role: "assistant", Content: snapshot})
		s.lastActive = time.Now()
		p.mu.Unlock()
	}

	fs := firestore.Get()
	r := models.NewLLMResponse(requestID, sessionID, data.Channel, data.Nick, p.cfg.Proxy.Ollama.Model, data.Prompt, snapshot, complete)
	if err = fs.CreateLLMResponse(r); err != nil {
		return fmt.Errorf("error saving LLM response to firestore: %w", err)
	}

	logger.Debugf(nil, "LLM response saved to firestore for %s in %s [complete: %v]", data.Nick, data.Channel, complete)

	if err = p.publishResponse(requestID, data.Channel, data.Nick, r.ID, sessionID, !complete); err != nil {
		return err
	}

	// If still streaming, update Firestore and session history when done
	if !complete {
		go func() {
			<-streamDone
			contentMu.Lock()
			final := strings.TrimSpace(thinkPattern.ReplaceAllString(content.String(), ""))
			contentMu.Unlock()

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

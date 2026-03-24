package main

import (
	"assistant/pkg/log"
	"assistant/pkg/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const defaultSessionTimeout = 10 * time.Minute

var thinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

type session struct {
	messages   []ollamaMessage
	lastActive time.Time
}

func sessionKey(channel, nick string) string {
	return channel + ":" + nick
}

func (p *proxy) sessionTimeout() time.Duration {
	if p.cfg.Proxy.Ollama.SessionTimeout != "" {
		if d, err := time.ParseDuration(p.cfg.Proxy.Ollama.SessionTimeout); err == nil {
			return d
		}
	}
	return defaultSessionTimeout
}

func (p *proxy) getOrCreateSession(channel, nick string) (*session, []ollamaMessage) {
	key := sessionKey(channel, nick)
	timeout := p.sessionTimeout()

	p.mu.Lock()
	defer p.mu.Unlock()

	s, ok := p.sessions[key]
	if !ok || time.Since(s.lastActive) > timeout {
		s = &session{}
		p.sessions[key] = s
	}
	return s, append([]ollamaMessage{}, s.messages...)
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

func (p *proxy) handleLLM(requestID string, data models.ProxyLLMRequestTaskData) error {
	logger := log.Logger()
	logger.Debugf(nil, "LLM request from %s in %s: %s", data.Nick, data.Channel, data.Prompt)

	s, history := p.getOrCreateSession(data.Channel, data.Nick)

	messages := []ollamaMessage{}
	if p.cfg.Proxy.Ollama.Prompt != "" {
		prompt := strings.NewReplacer(
			"{nick}", p.cfg.IRC.Nick,
			"{channel}", data.Channel,
			"{server}", p.cfg.IRC.ServerName,
		).Replace(p.cfg.Proxy.Ollama.Prompt)
		messages = append(messages, ollamaMessage{Role: "system", Content: prompt})
	}
	messages = append(messages, history...)
	messages = append(messages, ollamaMessage{Role: "user", Content: data.Prompt})

	req := ollamaRequest{
		Model:    p.cfg.Proxy.Ollama.Model,
		Messages: messages,
		Stream:   false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling ollama request: %w", err)
	}

	resp, err := http.Post(p.cfg.Proxy.Ollama.Endpoint+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error calling ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading ollama response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err = json.Unmarshal(respBody, &ollamaResp); err != nil {
		return fmt.Errorf("error unmarshaling ollama response: %w", err)
	}

	content := strings.TrimSpace(thinkPattern.ReplaceAllString(ollamaResp.Message.Content, ""))
	if len(content) == 0 {
		return fmt.Errorf("empty response from ollama")
	}

	p.mu.Lock()
	s.messages = append(s.messages, ollamaMessage{Role: "user", Content: data.Prompt})
	s.messages = append(s.messages, ollamaMessage{Role: "assistant", Content: content})
	s.lastActive = time.Now()
	p.mu.Unlock()

	responseMessages := splitMessages(content)
	logger.Debugf(nil, "LLM response to %s in %s: %d message(s)", data.Nick, data.Channel, len(responseMessages))

	return p.publishResponse(requestID, data.Channel, data.Nick, responseMessages)
}

// splitMessages splits content on newlines, discarding blank lines.
func splitMessages(content string) []string {
	lines := strings.Split(content, "\n")
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); len(trimmed) > 0 {
			messages = append(messages, trimmed)
		}
	}
	return messages
}

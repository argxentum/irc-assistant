package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"fmt"
	"strings"
	"time"
)

const roastSystemPrompt = `You are a roast comedian on IRC. You will be given a user's recent messages and must deliver
a short, witty roast (2-3 sentences max) based on what they've been saying. Be funny and clever, not mean-spirited or
hateful. Avoid anything racist, sexist, homophobic, or otherwise bigoted. Keep it lighthearted — the goal is to make the
channel laugh, not to hurt anyone. Do not use hashtags or emojis. Respond with only the roast, no preamble.`

func (p *proxy) handleRoast(requestID string, data models.ProxyLLMRequestTaskData) error {
	logger := log.Logger()
	logger.Debugf(nil, "roast request from %s in %s", data.Nick, data.Channel)

	messages := []ollamaMessage{
		{Role: "system", Content: strings.ReplaceAll(roastSystemPrompt, "\n", " ")},
		{Role: "user", Content: data.Prompt},
	}

	options := map[string]any{"num_predict": 256, "temperature": 0.8, "num_ctx": 4096}
	start := time.Now()

	sr, err := p.streamOllamaChat(messages, options, start)
	if err != nil {
		return err
	}

	// Wait for completion since roasts are short
	<-sr.done
	sr.complete = true

	snapshot := sr.snapshot()
	snapshot = strings.TrimSpace(snapshot)

	if len(snapshot) == 0 {
		return fmt.Errorf("empty roast response from ollama")
	}

	fs := firestore.Get()
	r := models.NewLLMResponse(requestID, "", data.Channel, data.Nick, p.cfg.Proxy.Ollama.Model, data.Prompt, snapshot, true)
	if err = fs.CreateLLMResponse(r); err != nil {
		return fmt.Errorf("error saving roast response to firestore: %w", err)
	}

	return p.publishResponse(requestID, data.Channel, data.Nick, r.ID, "", false)
}

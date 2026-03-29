package main

import (
	"assistant/pkg/api/trivia"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
)

func (s *server) triviaSetupHandler(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	if channel == "" {
		http.Error(w, "channel is required", http.StatusBadRequest)
		return
	}

	decoded, err := url.PathUnescape(channel)
	if err == nil {
		channel = decoded
	}

	t, err := template.ParseFiles(templatesRoot + "/trivia.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing template: %v", err), http.StatusInternalServerError)
		return
	}

	args := map[string]any{
		"channel":      channel,
		"categories":   trivia.Categories,
		"maxQuestions":  s.cfg.Trivia.MaxQuestions,
		"defaultCount": s.cfg.Trivia.DefaultCount,
	}

	if args["maxQuestions"] == 0 {
		args["maxQuestions"] = 9
	}
	if args["defaultCount"] == 0 {
		args["defaultCount"] = 5
	}

	err = t.Execute(w, args)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
	}
}

func (s *server) triviaStartHandler(w http.ResponseWriter, r *http.Request) {
	logger := log.Logger()

	channel := r.FormValue("channel")
	category := r.FormValue("category")
	difficulty := r.FormValue("difficulty")
	countStr := r.FormValue("count")
	firstAnswerOnly := r.FormValue("first_answer_only") == "true"

	if channel == "" {
		http.Error(w, "channel is required", http.StatusBadRequest)
		return
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 {
		count = 5
	}

	maxQuestions := s.cfg.Trivia.MaxQuestions
	if maxQuestions <= 0 {
		maxQuestions = 9
	}
	if count > maxQuestions {
		count = maxQuestions
	}

	questions, err := trivia.FetchQuestions(count, category, difficulty)
	if err != nil {
		logger.Errorf(nil, "error fetching trivia questions: %s", err)
		http.Error(w, "Failed to fetch trivia questions. Try again later.", http.StatusInternalServerError)
		return
	}

	if len(questions) == 0 {
		http.Error(w, "No trivia questions available. Try again later.", http.StatusServiceUnavailable)
		return
	}

	taskQuestions := make([]models.TriviaQuestion, 0, len(questions))
	for _, q := range questions {
		taskQuestions = append(taskQuestions, models.TriviaQuestion{
			Question:     q.Question,
			Answers:      q.Answers,
			CorrectIndex: q.CorrectIndex,
			Category:     q.Category,
			Difficulty:   q.Difficulty,
		})
	}

	task := models.NewTriviaStartTask(channel, taskQuestions, firstAnswerOnly)
	if err := queue.GetDefault().Publish(task); err != nil {
		logger.Errorf(nil, "error publishing trivia start task: %s", err)
		http.Error(w, "Failed to start trivia game. Try again later.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><script src="https://unpkg.com/@tailwindcss/browser@4"></script></head><body class="bg-gray-900 text-white flex items-center justify-center min-h-screen"><div class="text-center"><h1 class="text-2xl font-bold mb-4">🎯 Trivia Started!</h1><p class="text-gray-400">%d questions have been sent to %s. Head back to IRC!</p></div></body></html>`, len(questions), template.HTMLEscapeString(channel))
}

package models

import "time"

type TriviaStartTaskData struct {
	Channel         string           `json:"channel"`
	StartedBy       string           `json:"started_by"`
	Questions       []TriviaQuestion `json:"questions"`
	FirstAnswerOnly bool             `json:"first_answer_only"`
}

type TriviaQuestion struct {
	Question     string   `json:"question"`
	Answers      []string `json:"answers"`
	CorrectIndex int      `json:"correct_index"`
	Category     string   `json:"category"`
	Difficulty   string   `json:"difficulty"`
}

func NewTriviaStartTask(channel, startedBy string, questions []TriviaQuestion, firstAnswerOnly bool) *Task {
	return newTask(TaskTypeTriviaStart, time.Now(), TriviaStartTaskData{
		Channel:         channel,
		StartedBy:       startedBy,
		Questions:       questions,
		FirstAnswerOnly: firstAnswerOnly,
	})
}

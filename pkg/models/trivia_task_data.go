package models

import "time"

type TriviaStartTaskData struct {
	Channel         string           `json:"channel"`
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

func NewTriviaStartTask(channel string, questions []TriviaQuestion, firstAnswerOnly bool) *Task {
	return newTask(TaskTypeTriviaStart, time.Now(), TriviaStartTaskData{
		Channel:         channel,
		Questions:       questions,
		FirstAnswerOnly: firstAnswerOnly,
	})
}

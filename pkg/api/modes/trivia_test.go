package modes

import (
	"assistant/pkg/api/trivia"
	"testing"
)

func TestNewTriviaModeRejectsInvalidQuestions(t *testing.T) {
	tests := []struct {
		name      string
		questions []trivia.Question
	}{
		{name: "empty"},
		{
			name: "no answers",
			questions: []trivia.Question{{
				Question:     "Question?",
				CorrectIndex: 1,
			}},
		},
		{
			name: "correct answer below range",
			questions: []trivia.Question{{
				Question:     "Question?",
				Answers:      []string{"Answer"},
				CorrectIndex: 0,
			}},
		},
		{
			name: "correct answer above range",
			questions: []trivia.Question{{
				Question:     "Question?",
				Answers:      []string{"Answer"},
				CorrectIndex: 2,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewTriviaMode("#channel", nil, nil, tt.questions); err == nil {
				t.Fatal("NewTriviaMode accepted invalid questions")
			}
		})
	}
}

func TestNewTriviaModeAcceptsValidQuestions(t *testing.T) {
	questions := []trivia.Question{{
		Question:     "Question?",
		Answers:      []string{"Wrong", "Correct"},
		CorrectIndex: 2,
	}}

	mode, err := NewTriviaMode("#channel", nil, nil, questions)
	if err != nil {
		t.Fatalf("NewTriviaMode returned error: %v", err)
	}
	if mode == nil {
		t.Fatal("NewTriviaMode returned nil mode")
	}
}

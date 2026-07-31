package queue

import (
	"assistant/pkg/models"
	"errors"
	"testing"
	"time"
)

func TestProcessMessage(t *testing.T) {
	task := models.NewReminderTask(time.Now(), "nick", "#channel", "remember this")
	data, err := task.Serialize()
	if err != nil {
		t.Fatalf("serialize task: %v", err)
	}

	t.Run("successful callback", func(t *testing.T) {
		called := false
		err := processMessage(data, func(received *models.Task) error {
			called = true
			if received.ID != task.ID {
				t.Fatalf("received task %q, want %q", received.ID, task.ID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("process message: %v", err)
		}
		if !called {
			t.Fatal("callback was not called")
		}
	})

	t.Run("callback failure", func(t *testing.T) {
		callbackErr := errors.New("temporary failure")
		err := processMessage(data, func(*models.Task) error {
			return callbackErr
		})
		if !errors.Is(err, callbackErr) {
			t.Fatalf("process message error = %v, want %v", err, callbackErr)
		}
	})

	t.Run("malformed message", func(t *testing.T) {
		called := false
		err := processMessage([]byte("{"), func(*models.Task) error {
			called = true
			return nil
		})
		if err == nil {
			t.Fatal("process message succeeded, want deserialization error")
		}
		if called {
			t.Fatal("callback was called for malformed message")
		}
	})
}

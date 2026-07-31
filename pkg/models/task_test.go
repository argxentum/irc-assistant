package models

import (
	"testing"
	"time"
)

func TestTaskIsDurable(t *testing.T) {
	durableTypes := []string{
		TaskTypeReminder,
		TaskTypeBanRemoval,
		TaskTypeMuteRemoval,
		TaskTypeDisinformationMutePenaltyRemoval,
		TaskTypeDisinformationBanPenaltyRemoval,
	}
	for _, taskType := range durableTypes {
		t.Run(taskType, func(t *testing.T) {
			if !(&Task{Type: taskType}).IsDurable() {
				t.Fatalf("task type %s is not durable", taskType)
			}
		})
	}

	ephemeralTypes := []string{
		TaskTypeReconnect,
		TaskTypeNotifyVoiceRequests,
		TaskTypePersistentChannel,
		TaskTypeProxyLLMRequest,
		TaskTypeProxyLLMResponse,
		TaskTypeProxySummaryRequest,
		TaskTypeProxySummaryResponse,
		TaskTypeProxyInactivityRequest,
		TaskTypeProxyInactivityResponse,
		TaskTypeProxyRedditSearchRequest,
		TaskTypeProxyRedditSearchResponse,
		TaskTypeDashboardResponse,
		TaskTypePersistentChannelStats,
		TaskTypeTriviaStart,
	}
	for _, taskType := range ephemeralTypes {
		t.Run(taskType, func(t *testing.T) {
			if (&Task{Type: taskType}).IsDurable() {
				t.Fatalf("task type %s is durable", taskType)
			}
		})
	}
}

func TestDashboardTaskIsDurable(t *testing.T) {
	durableActions := []string{
		DashboardActionMute,
		DashboardActionUnban,
		DashboardActionUnmute,
		DashboardActionExpireBan,
		DashboardActionExpireMute,
		DashboardActionApproveVR,
	}
	for _, action := range durableActions {
		t.Run(action, func(t *testing.T) {
			task := &Task{
				Type: TaskTypeDashboardRequest,
				Data: DashboardRequestTaskData{Action: action},
			}
			if !task.IsDurable() {
				t.Fatalf("dashboard action %s is not durable", action)
			}
		})
	}

	ephemeralActions := []string{
		DashboardActionListUsers,
		DashboardActionKick,
		DashboardActionBan,
		DashboardActionAddBan,
		DashboardActionListBans,
		DashboardActionGetTopic,
		DashboardActionSetTopic,
		DashboardActionDenyVR,
		DashboardActionListCommands,
	}
	for _, action := range ephemeralActions {
		t.Run(action, func(t *testing.T) {
			task := &Task{
				Type: TaskTypeDashboardRequest,
				Data: DashboardRequestTaskData{Action: action},
			}
			if task.IsDurable() {
				t.Fatalf("dashboard action %s is durable", action)
			}
		})
	}
}

func TestDeserializedDashboardTaskRetainsDurability(t *testing.T) {
	task := NewDashboardRequestTask("request-id", DashboardRequestTaskData{Action: DashboardActionUnmute})
	serialized, err := task.Serialize()
	if err != nil {
		t.Fatalf("serialize task: %v", err)
	}

	deserialized, err := DeserializeTask(serialized)
	if err != nil {
		t.Fatalf("deserialize task: %v", err)
	}
	if !deserialized.IsDurable() {
		t.Fatal("deserialized unmute request is not durable")
	}
}

func TestTaskIsStaleAtStartup(t *testing.T) {
	startedAt := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		task *Task
		want bool
	}{
		{
			name: "old ephemeral task",
			task: &Task{Type: TaskTypeProxyLLMRequest, CreatedAt: startedAt.Add(-time.Second), DueAt: startedAt.Add(-time.Second)},
			want: true,
		},
		{
			name: "new ephemeral task",
			task: &Task{Type: TaskTypeProxyLLMRequest, CreatedAt: startedAt.Add(time.Second), DueAt: startedAt.Add(time.Second)},
			want: false,
		},
		{
			name: "scheduled after startup",
			task: &Task{Type: TaskTypeNotifyVoiceRequests, CreatedAt: startedAt.Add(-time.Hour), DueAt: startedAt.Add(time.Hour)},
			want: false,
		},
		{
			name: "old recurring occurrence",
			task: &Task{Type: TaskTypePersistentChannel, DueAt: startedAt.Add(-time.Minute)},
			want: true,
		},
		{
			name: "old durable task",
			task: &Task{Type: TaskTypeReminder, CreatedAt: startedAt.Add(-time.Hour), DueAt: startedAt.Add(-time.Minute)},
			want: false,
		},
		{
			name: "old durable dashboard voice task",
			task: &Task{
				Type:      TaskTypeDashboardRequest,
				CreatedAt: startedAt.Add(-time.Hour),
				DueAt:     startedAt.Add(-time.Hour),
				Data:      DashboardRequestTaskData{Action: DashboardActionUnmute},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.IsStaleAtStartup(startedAt); got != tt.want {
				t.Fatalf("IsStaleAtStartup() = %t, want %t", got, tt.want)
			}
		})
	}
}

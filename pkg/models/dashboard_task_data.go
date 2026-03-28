package models

import "time"

const (
	DashboardActionListUsers  = "list_users"
	DashboardActionKick       = "kick"
	DashboardActionBan        = "ban"
	DashboardActionMute       = "mute"
	DashboardActionUnban      = "unban"
	DashboardActionUnmute     = "unmute"
	DashboardActionExpireBan  = "expire_ban"
	DashboardActionExpireMute = "expire_mute"
)

type DashboardRequestTaskData struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Channel   string `json:"channel"`
	Nick      string `json:"nick,omitempty"`
	Mask      string `json:"mask,omitempty"`
	Duration  string `json:"duration,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type DashboardResponseTaskData struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type DashboardUser struct {
	Nick   string `json:"nick"`
	User   string `json:"user"`
	Host   string `json:"host"`
	Status string `json:"status"`
}

func NewDashboardRequestTask(requestID string, data DashboardRequestTaskData) *Task {
	data.RequestID = requestID
	return &Task{
		ID:        requestID,
		Type:      TaskTypeDashboardRequest,
		CreatedAt: time.Now(),
		DueAt:     time.Now(),
		Status:    TaskStatusPending,
		Data:      data,
	}
}

func NewDashboardResponseTask(requestID, action string, success bool, errMsg string, data any) *Task {
	return &Task{
		ID:        "dash-resp-" + requestID,
		Type:      TaskTypeDashboardResponse,
		CreatedAt: time.Now(),
		DueAt:     time.Now(),
		Status:    TaskStatusPending,
		Data: DashboardResponseTaskData{
			RequestID: requestID,
			Action:    action,
			Success:   success,
			Error:     errMsg,
			Data:      data,
		},
	}
}

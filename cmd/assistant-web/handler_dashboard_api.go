package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"encoding/json"
	"net/http"
)

func (s *server) dashboardUsersHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionListUsers,
		Channel: session.Channel,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard users request failed: %s", err)
		http.Error(w, "Request failed", http.StatusGatewayTimeout)
		return
	}

	if !resp.Success {
		http.Error(w, resp.Error, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Data)
}

type dashboardActionRequest struct {
	Nick     string `json:"nick"`
	Duration string `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func (s *server) dashboardActionHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	action := r.PathValue("action")

	var req dashboardActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Nick == "" {
		http.Error(w, "Nick is required", http.StatusBadRequest)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:   action,
		Channel:  session.Channel,
		Nick:     req.Nick,
		Duration: req.Duration,
		Reason:   req.Reason,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard action %s failed: %s", action, err)
		http.Error(w, "Request failed", http.StatusGatewayTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *server) dashboardUsersByHostHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	host := r.PathValue("host")
	if host == "" {
		http.Error(w, "Host is required", http.StatusBadRequest)
		return
	}

	users, err := firestore.Get().GetUsersByHost(session.Channel, host)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard users by host query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type hostUser struct {
		Nick      string `json:"nick"`
		UserID    string `json:"user_id"`
		Host      string `json:"host"`
		CreatedAt int64  `json:"created_at"`
		UpdatedAt int64  `json:"updated_at"`
	}

	result := make([]hostUser, 0, len(users))
	for _, u := range users {
		result = append(result, hostUser{
			Nick:      u.Nick,
			UserID:    u.UserID,
			Host:      u.Host,
			CreatedAt: u.CreatedAt.Unix(),
			UpdatedAt: u.UpdatedAt.Unix(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

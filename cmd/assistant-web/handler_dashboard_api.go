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

func (s *server) dashboardAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	users, err := firestore.Get().GetAllUsers(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard all users query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type allUser struct {
		Nick        string  `json:"nick"`
		Host        string  `json:"host"`
		IsAutoVoiced bool   `json:"is_auto_voiced"`
		Karma       int     `json:"karma"`
		UpdatedAt   int64   `json:"updated_at"`
	}

	result := make([]allUser, 0, len(users))
	for _, u := range users {
		result = append(result, allUser{
			Nick:         u.Nick,
			Host:         u.Host,
			IsAutoVoiced: u.IsAutoVoiced,
			Karma:        u.Karma,
			UpdatedAt:    u.UpdatedAt.Unix(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

	if action == "autovoice" {
		s.handleAutoVoiceAction(w, session.Channel, req.Nick, true)
		return
	}

	if action == "removeautovoice" {
		s.handleAutoVoiceAction(w, session.Channel, req.Nick, false)
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

func (s *server) handleAutoVoiceAction(w http.ResponseWriter, channel, nick string, enable bool) {
	fs := firestore.Get()
	user, err := fs.GetUserByNick(channel, nick)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "user not found"})
		return
	}

	err = fs.UpdateUser(channel, user, map[string]any{"is_auto_voiced": enable})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard auto-voice failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "update failed"})
		return
	}

	if enable {
		// also voice the user via IRC
		resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
			Action:  models.DashboardActionUnmute,
			Channel: channel,
			Nick:    nick,
		})
		if err != nil {
			log.Logger().Warningf(nil, "dashboard auto-voice unmute failed: %s", err)
		} else if !resp.Success {
			log.Logger().Warningf(nil, "dashboard auto-voice unmute failed: %s", resp.Error)
		}
	}

	log.Logger().Infof(nil, "dashboard: set auto-voice %v for %s in %s", enable, nick, channel)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardUserHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	nick := r.PathValue("nick")
	if nick == "" {
		http.Error(w, "Nick is required", http.StatusBadRequest)
		return
	}

	user, err := firestore.Get().GetUserByNick(session.Channel, nick)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard user query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	type recentMessage struct {
		Message string `json:"message"`
		At      int64  `json:"at"`
	}

	messages := make([]recentMessage, 0, len(user.RecentMessages))
	for _, m := range user.RecentMessages {
		messages = append(messages, recentMessage{
			Message: m.Message,
			At:      m.At.Unix(),
		})
	}

	result := map[string]any{
		"nick":            user.Nick,
		"user_id":         user.UserID,
		"host":            user.Host,
		"karma":           user.Karma,
		"penalty":         user.Penalty,
		"location":        user.Location,
		"is_auto_voiced":  user.IsAutoVoiced,
		"credibility":     credibilityScore(user),
		"recent_messages": messages,
		"created_at":      user.CreatedAt.Unix(),
		"updated_at":      user.UpdatedAt.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func credibilityScore(user *models.User) *float64 {
	total := user.HighCredibilityCount + user.LowCredibilityCount
	if total == 0 {
		return nil
	}
	score := float64(user.HighCredibilityCount) / float64(total) * 100
	return &score
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

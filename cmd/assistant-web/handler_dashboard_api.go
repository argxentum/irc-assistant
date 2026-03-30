package main

import (
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/penalty"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		Nick         string `json:"nick"`
		Host         string `json:"host"`
		IsAutoVoiced bool   `json:"is_auto_voiced"`
		Karma        int    `json:"karma"`
		UpdatedAt    int64  `json:"updated_at"`
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
	Nick        string `json:"nick"`
	Duration    string `json:"duration,omitempty"`
	Reason      string `json:"reason,omitempty"`
	IncludeHost bool   `json:"include_host,omitempty"`
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
		s.handleAutoVoiceAction(w, session.Channel, req.Nick, true, req.IncludeHost)
		return
	}

	if action == "removeautovoice" {
		s.handleAutoVoiceAction(w, session.Channel, req.Nick, false, req.IncludeHost)
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

func (s *server) handleAutoVoiceAction(w http.ResponseWriter, channel, nick string, enable, includeHost bool) {
	logger := log.Logger()
	fs := firestore.Get()

	user, err := fs.GetUserByNick(channel, nick)
	if err != nil || user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "user not found"})
		return
	}

	err = fs.UpdateUser(channel, user, map[string]any{"is_auto_voiced": enable})
	if err != nil {
		logger.Errorf(nil, "dashboard auto-voice failed: %s", err)
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
			logger.Warningf(nil, "dashboard auto-voice unmute failed: %s", err)
		} else if !resp.Success {
			logger.Warningf(nil, "dashboard auto-voice unmute failed: %s", resp.Error)
		}
	}

	// apply to all users with the same host if requested
	if includeHost && user.Host != "" {
		hostUsers, err := fs.GetUsersByHost(channel, user.Host)
		if err != nil {
			logger.Warningf(nil, "dashboard auto-voice host lookup failed: %s", err)
		} else {
			for _, hu := range hostUsers {
				if hu.Nick == nick {
					continue
				}
				if err := fs.UpdateUser(channel, hu, map[string]any{"is_auto_voiced": enable}); err != nil {
					logger.Warningf(nil, "dashboard auto-voice host update failed for %s: %s", hu.Nick, err)
					continue
				}
				if enable {
					resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
						Action:  models.DashboardActionUnmute,
						Channel: channel,
						Nick:    hu.Nick,
					})
					if err != nil {
						logger.Warningf(nil, "dashboard auto-voice unmute failed for %s: %s", hu.Nick, err)
					} else if !resp.Success {
						logger.Warningf(nil, "dashboard auto-voice unmute failed for %s: %s", hu.Nick, resp.Error)
					}
				}
				logger.Infof(nil, "dashboard: set auto-voice %v for %s (host match) in %s", enable, hu.Nick, channel)
			}
		}
	}

	logger.Infof(nil, "dashboard: set auto-voice %v for %s in %s", enable, nick, channel)

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
		"nick":             user.Nick,
		"user_id":          user.UserID,
		"host":             user.Host,
		"karma":            user.Karma,
		"penalty":          user.Penalty,
		"extended_penalty": user.ExtendedPenalty,
		"penalty_status":   penalty.Calculate(user.Penalty, user.ExtendedPenalty, s.cfg.DisinfoPenalty),
		"location":         user.Location,
		"is_auto_voiced":   user.IsAutoVoiced,
		"credibility":      credibilityScore(user),
		"recent_messages":  messages,
		"created_at":       user.CreatedAt.Unix(),
		"updated_at":       user.UpdatedAt.Unix(),
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

func (s *server) dashboardBansHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionListBans,
		Channel: session.Channel,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard bans request failed: %s", err)
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

func (s *server) dashboardVoiceRequestsHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fs := firestore.Get()
	ch, err := fs.Channel(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard voice requests: error getting channel: %s", err)
		http.Error(w, "Failed to get channel", http.StatusInternalServerError)
		return
	}

	type voiceRequest struct {
		Nick        string `json:"nick"`
		Username    string `json:"username"`
		Host        string `json:"host"`
		RequestedAt int64  `json:"requested_at"`
	}

	result := make([]voiceRequest, 0)
	if ch != nil {
		for _, vr := range ch.VoiceRequests {
			result = append(result, voiceRequest{
				Nick:        vr.Nick,
				Username:    vr.Username,
				Host:        vr.Host,
				RequestedAt: vr.RequestedAt.Unix(),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardVoiceRequestActionHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	action := r.PathValue("action")
	if action != "approve" && action != "deny" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	var req struct {
		Nick string `json:"nick"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nick == "" {
		http.Error(w, "Nick is required", http.StatusBadRequest)
		return
	}

	dashAction := models.DashboardActionApproveVR
	if action == "deny" {
		dashAction = models.DashboardActionDenyVR
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  dashAction,
		Channel: session.Channel,
		Nick:    req.Nick,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard voice request %s failed: %s", action, err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "action failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": resp.Success, "error": resp.Error})
}

func (s *server) dashboardGetTopicHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionGetTopic,
		Channel: session.Channel,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard get topic failed: %s", err)
		http.Error(w, "Request failed", http.StatusGatewayTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": resp.Success, "topic": resp.Data})
}

func (s *server) dashboardSetTopicHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Topic string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionSetTopic,
		Channel: session.Channel,
		Topic:   req.Topic,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard set topic failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "action failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": resp.Success, "error": resp.Error})
}

func (s *server) dashboardAddBanHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Mask string `json:"mask"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Mask == "" {
		http.Error(w, "Mask is required", http.StatusBadRequest)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionAddBan,
		Channel: session.Channel,
		Mask:    req.Mask,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard add ban failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "action failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": resp.Success, "error": resp.Error})
}

func (s *server) dashboardRemoveBanHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Mask string `json:"mask"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Mask == "" {
		http.Error(w, "Mask is required", http.StatusBadRequest)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionExpireBan,
		Channel: session.Channel,
		Mask:    req.Mask,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard remove ban failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "action failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": resp.Success, "error": resp.Error})
}

func (s *server) dashboardPenaltiesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fs := firestore.Get()

	type penalty struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Nick  string `json:"nick,omitempty"`
		Mask  string `json:"mask,omitempty"`
		Host  string `json:"host,omitempty"`
		DueAt int64  `json:"due_at"`
	}

	var penalties []penalty

	bans, err := fs.GetPendingTasks("", session.Channel, models.TaskTypeBanRemoval)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard penalties: error getting bans: %s", err)
	} else {
		for _, t := range bans {
			data := t.Data.(models.BanRemovalTaskData)
			penalties = append(penalties, penalty{
				ID:    t.ID,
				Type:  "ban",
				Mask:  data.Mask,
				DueAt: t.DueAt.Unix(),
			})
		}
	}

	mutes, err := fs.GetPendingTasks("", session.Channel, models.TaskTypeMuteRemoval)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard penalties: error getting mutes: %s", err)
	} else {
		for _, t := range mutes {
			data := t.Data.(models.MuteRemovalTaskData)
			penalties = append(penalties, penalty{
				ID:    t.ID,
				Type:  "mute",
				Nick:  data.Nick,
				Host:  data.Host,
				DueAt: t.DueAt.Unix(),
			})
		}
	}

	if penalties == nil {
		penalties = []penalty{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(penalties)
}

func (s *server) dashboardExpirePenaltyHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fs := firestore.Get()
	var taskType string

	switch req.Type {
	case "ban":
		taskType = models.TaskTypeBanRemoval
	case "mute":
		taskType = models.TaskTypeMuteRemoval
	default:
		http.Error(w, "Invalid penalty type", http.StatusBadRequest)
		return
	}

	tasks, err := fs.GetPendingTasks("", session.Channel, taskType)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard expire penalty: error getting tasks: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "failed to find task"})
		return
	}

	var task *models.Task
	for _, t := range tasks {
		if t.ID == req.ID {
			task = t
			break
		}
	}

	if task == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "task not found"})
		return
	}

	// execute the removal action via IRC
	reqData := models.DashboardRequestTaskData{
		Channel: session.Channel,
	}
	if req.Type == "ban" {
		data := task.Data.(models.BanRemovalTaskData)
		reqData.Action = models.DashboardActionExpireBan
		reqData.Mask = data.Mask
	} else {
		data := task.Data.(models.MuteRemovalTaskData)
		reqData.Action = models.DashboardActionExpireMute
		reqData.Nick = data.Nick
	}

	resp, err := s.dashboardRequest(reqData)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard expire penalty action failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "action failed"})
		return
	}

	if !resp.Success {
		log.Logger().Warningf(nil, "dashboard expire penalty: IRC action failed for %s: %s", req.ID, resp.Error)
	}

	// cancel the scheduled task
	task.Status = models.TaskStatusCancelled
	if err := fs.CompleteTask(task); err != nil {
		log.Logger().Errorf(nil, "dashboard expire penalty: error cancelling task %s: %s", req.ID, err)
	}

	log.Logger().Infof(nil, "dashboard: expired %s penalty %s", req.Type, req.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardUsersByMaskHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mask := r.PathValue("mask")
	if mask == "" {
		http.Error(w, "Mask is required", http.StatusBadRequest)
		return
	}

	// parse nick!userid@host
	var nick, userID, host string
	if atIdx := strings.LastIndex(mask, "@"); atIdx >= 0 {
		host = mask[atIdx+1:]
		left := mask[:atIdx]
		if bangIdx := strings.Index(left, "!"); bangIdx >= 0 {
			nick = left[:bangIdx]
			userID = left[bangIdx+1:]
		} else {
			nick = left
		}
	} else {
		nick = mask
	}

	users, err := firestore.Get().GetUsersByMask(session.Channel, nick, userID, host)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard users by mask query failed: %s", err)
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

func (s *server) dashboardSourcesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sources, err := firestore.Get().ListSources()
	if err != nil {
		log.Logger().Errorf(nil, "dashboard sources query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type sourceItem struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Bias        string   `json:"bias"`
		Factuality  string   `json:"factuality"`
		Credibility string   `json:"credibility"`
		Reviews     []string `json:"reviews"`
		URLs        []string `json:"urls"`
		Paywall     bool     `json:"paywall"`
		Keywords    []string `json:"keywords"`
		Citations   int      `json:"citations"`
		CreatedAt   int64    `json:"created_at"`
		UpdatedAt   int64    `json:"updated_at"`
	}

	result := make([]sourceItem, 0, len(sources))
	for _, src := range sources {
		result = append(result, sourceItem{
			ID:          src.ID,
			Title:       src.Title,
			Bias:        src.Bias,
			Factuality:  src.Factuality,
			Credibility: src.Credibility,
			Reviews:     src.Reviews,
			URLs:        src.URLs,
			Paywall:     src.Paywall,
			Keywords:    src.Keywords,
			Citations:   src.Citations,
			CreatedAt:   src.CreatedAt.Unix(),
			UpdatedAt:   src.UpdatedAt.Unix(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardSourceHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	source, err := firestore.Get().GetSource(id)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard source query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	if source == nil {
		http.Error(w, "Source not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source)
}

func (s *server) dashboardSourceSaveHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Bias        string   `json:"bias"`
		Factuality  string   `json:"factuality"`
		Credibility string   `json:"credibility"`
		Reviews     []string `json:"reviews"`
		URLs        []string `json:"urls"`
		Paywall     bool     `json:"paywall"`
		Keywords    []string `json:"keywords"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	fs := firestore.Get()

	if req.ID == "" {
		// create new source
		source := models.NewEmptySource()
		source.Title = req.Title
		source.Bias = req.Bias
		source.Factuality = req.Factuality
		source.Credibility = req.Credibility
		source.Reviews = req.Reviews
		source.URLs = req.URLs
		source.Paywall = req.Paywall
		source.Keywords = req.Keywords

		if source.Reviews == nil {
			source.Reviews = make([]string, 0)
		}
		if source.URLs == nil {
			source.URLs = make([]string, 0)
		}
		if source.Keywords == nil {
			source.Keywords = make([]string, 0)
		}

		if err := fs.CreateSource(source); err != nil {
			log.Logger().Errorf(nil, "dashboard create source failed: %s", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "create failed"})
			return
		}

		// remove any unknown source records matching the new source's URLs
		for _, u := range source.URLs {
			if err := fs.DeleteUnknownSource(u); err != nil {
				log.Logger().Debugf(nil, "no unknown source to clean up for %s", u)
			}
		}

		log.Logger().Infof(nil, "dashboard: created source %s (%s)", source.ID, source.Title)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": true, "id": source.ID})
		return
	}

	// update existing source
	existing, err := fs.GetSource(req.ID)
	if err != nil || existing == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "source not found"})
		return
	}

	existing.Title = req.Title
	existing.Bias = req.Bias
	existing.Factuality = req.Factuality
	existing.Credibility = req.Credibility
	existing.Reviews = req.Reviews
	existing.URLs = req.URLs
	existing.Paywall = req.Paywall
	existing.Keywords = req.Keywords

	if existing.Reviews == nil {
		existing.Reviews = make([]string, 0)
	}
	if existing.URLs == nil {
		existing.URLs = make([]string, 0)
	}
	if existing.Keywords == nil {
		existing.Keywords = make([]string, 0)
	}

	if err := fs.SetSource(existing); err != nil {
		log.Logger().Errorf(nil, "dashboard update source failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "update failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: updated source %s (%s)", existing.ID, existing.Title)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardSourceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().DeleteSource(req.ID); err != nil {
		log.Logger().Errorf(nil, "dashboard delete source failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "delete failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: deleted source %s", req.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardTopSourcesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sources, err := firestore.Get().ListSources()
	if err != nil {
		log.Logger().Errorf(nil, "dashboard top sources query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type topSource struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Credibility string `json:"credibility"`
		Citations   int    `json:"citations"`
	}

	// filter to sources with citations and sort desc
	result := make([]topSource, 0)
	for _, src := range sources {
		if src.Citations > 0 {
			result = append(result, topSource{
				ID:          src.ID,
				Title:       src.Title,
				Credibility: src.Credibility,
				Citations:   src.Citations,
			})
		}
	}

	// sort by citations desc
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Citations > result[i].Citations {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardUnknownSourcesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	unmatched, err := firestore.Get().ListUnknownSources()
	if err != nil {
		log.Logger().Errorf(nil, "dashboard unknown sources query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type unknownItem struct {
		Domain    string `json:"domain"`
		Citations int    `json:"citations"`
	}

	result := make([]unknownItem, 0, len(unmatched))
	for _, u := range unmatched {
		result = append(result, unknownItem{
			Domain:    u.Domain,
			Citations: u.Citations,
		})
	}

	// sort by citations desc
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Citations > result[i].Citations {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardBannedWordsHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	words, err := firestore.Get().BannedWords(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "dashboard banned words query failed: %s", err)
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}

	type bannedWord struct {
		Word string `json:"word"`
	}

	result := make([]bannedWord, 0, len(words))
	for _, bw := range words {
		result = append(result, bannedWord{Word: bw.Word})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardBannedWordAddHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Word string `json:"word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Word == "" {
		http.Error(w, "Word is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().AddBannedWord(session.Channel, req.Word); err != nil {
		log.Logger().Errorf(nil, "dashboard add banned word failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "add failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: added banned word in %s", session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardBannedWordRemoveHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Word string `json:"word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Word == "" {
		http.Error(w, "Word is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().RemoveBannedWord(session.Channel, req.Word); err != nil {
		log.Logger().Errorf(nil, "dashboard remove banned word failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "remove failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: removed banned word in %s", session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 && parsed <= 168 {
			hours = parsed
		}
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	fs := firestore.Get()
	stats, err := fs.GetChannelStats(session.Channel, since)
	if err != nil {
		log.Logger().Errorf(nil, "error getting channel stats: %s", err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *server) dashboardCommandsHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := s.dashboardRequest(models.DashboardRequestTaskData{
		Action:  models.DashboardActionListCommands,
		Channel: session.Channel,
	})
	if err != nil {
		log.Logger().Errorf(nil, "dashboard commands request failed: %s", err)
		http.Error(w, "Request failed", http.StatusGatewayTimeout)
		return
	}

	if !resp.Success {
		http.Error(w, resp.Error, http.StatusInternalServerError)
		return
	}

	// the response data comes back as []any after JSON round-trip; re-marshal to decode properly
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		http.Error(w, "Failed to process commands", http.StatusInternalServerError)
		return
	}
	var commands []models.CommandInfo
	if err := json.Unmarshal(raw, &commands); err != nil {
		http.Error(w, "Failed to decode commands", http.StatusInternalServerError)
		return
	}

	ch, err := firestore.Get().Channel(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "error getting channel: %s", err)
		http.Error(w, "Failed to get channel", http.StatusInternalServerError)
		return
	}

	disabled := make(map[string]bool)
	if ch != nil {
		for _, d := range ch.DisabledCommands {
			disabled[d] = true
		}
	}

	type commandResponse struct {
		Name         string   `json:"name"`
		Description  string   `json:"description"`
		Triggers     []string `json:"triggers"`
		Usages       []string `json:"usages"`
		RequiresAuth bool     `json:"requires_auth"`
		AllowDM      bool     `json:"allow_dm"`
		Enabled      bool     `json:"enabled"`
	}

	result := make([]commandResponse, 0, len(commands))
	for _, cmd := range commands {
		result = append(result, commandResponse{
			Name:         cmd.Name,
			Description:  cmd.Description,
			Triggers:     cmd.Triggers,
			Usages:       cmd.Usages,
			RequiresAuth: cmd.RequiresAuth,
			AllowDM:      cmd.AllowDM,
			Enabled:      !disabled[cmd.Name],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardCommandToggleHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	fs := firestore.Get()
	ch, err := fs.Channel(session.Channel)
	if err != nil || ch == nil {
		http.Error(w, "Failed to get channel", http.StatusInternalServerError)
		return
	}

	if req.Enabled {
		// remove from disabled list
		filtered := make([]string, 0)
		for _, d := range ch.DisabledCommands {
			if d != req.Name {
				filtered = append(filtered, d)
			}
		}
		ch.DisabledCommands = filtered
	} else {
		// add to disabled list if not already there
		found := false
		for _, d := range ch.DisabledCommands {
			if d == req.Name {
				found = true
				break
			}
		}
		if !found {
			ch.DisabledCommands = append(ch.DisabledCommands, req.Name)
		}
	}

	if err := fs.UpdateChannel(session.Channel, map[string]any{"disabled_commands": ch.DisabledCommands, "updated_at": time.Now()}); err != nil {
		log.Logger().Errorf(nil, "error updating channel disabled commands: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "update failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: toggled command %s to enabled=%v in %s", req.Name, req.Enabled, session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardCommandUsageHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	usage, err := firestore.Get().ListCommandUsage(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "error listing command usage: %s", err)
		http.Error(w, "Failed to list command usage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

func (s *server) dashboardDisinfoSourcesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sources, err := firestore.Get().DisinformationSources(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "error listing disinfo sources: %s", err)
		http.Error(w, "Failed to list sources", http.StatusInternalServerError)
		return
	}

	type disinfoResponse struct {
		Source    string `json:"source"`
		CreatedAt string `json:"created_at"`
	}

	result := make([]disinfoResponse, 0, len(sources))
	for _, src := range sources {
		result = append(result, disinfoResponse{
			Source:    src.Source,
			CreatedAt: src.CreatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardDisinfoSourceAddHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		http.Error(w, "Source is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().AddDisinformationSource(session.Channel, req.Source); err != nil {
		log.Logger().Errorf(nil, "dashboard add disinfo source failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "add failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: added disinfo source %s in %s", req.Source, session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardDisinfoSourceRemoveHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		http.Error(w, "Source is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().DeleteDisinformationSource(session.Channel, req.Source); err != nil {
		log.Logger().Errorf(nil, "dashboard remove disinfo source failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "remove failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: removed disinfo source %s from %s", req.Source, session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardCommunityNotesHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notes, err := firestore.Get().CommunityNotes(session.Channel)
	if err != nil {
		log.Logger().Errorf(nil, "error listing community notes: %s", err)
		http.Error(w, "Failed to list notes", http.StatusInternalServerError)
		return
	}

	type noteResponse struct {
		ID             string   `json:"id"`
		Content        string   `json:"content"`
		Author         string   `json:"author"`
		Sources        []string `json:"sources"`
		CounterSources []string `json:"counter_sources"`
		NotedAt        string   `json:"noted_at"`
	}

	result := make([]noteResponse, 0, len(notes))
	for _, n := range notes {
		result = append(result, noteResponse{
			ID:             n.ID,
			Content:        n.Content,
			Author:         n.Author,
			Sources:        n.Sources,
			CounterSources: n.CounterSources,
			NotedAt:        n.NotedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *server) dashboardCommunityNoteSaveHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID             string   `json:"id"`
		Content        string   `json:"content"`
		Author         string   `json:"author"`
		Sources        []string `json:"sources"`
		CounterSources []string `json:"counter_sources"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		note := models.NewCommunityNote(req.Content, "", req.Author)
		note.Sources = req.Sources
		note.CounterSources = req.CounterSources
		if err := firestore.Get().CreateCommunityNote(session.Channel, note); err != nil {
			log.Logger().Errorf(nil, "dashboard create community note failed: %s", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "create failed"})
			return
		}
		log.Logger().Infof(nil, "dashboard: created community note in %s", session.Channel)
	} else {
		existing, err := firestore.Get().CommunityNote(session.Channel, req.ID)
		if err != nil || existing == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "note not found"})
			return
		}
		existing.Content = req.Content
		existing.Author = req.Author
		existing.Sources = req.Sources
		existing.CounterSources = req.CounterSources
		if err := firestore.Get().SetCommunityNote(session.Channel, existing); err != nil {
			log.Logger().Errorf(nil, "dashboard update community note failed: %s", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "update failed"})
			return
		}
		log.Logger().Infof(nil, "dashboard: updated community note %s in %s", req.ID, session.Channel)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *server) dashboardCommunityNoteDeleteHandler(w http.ResponseWriter, r *http.Request) {
	session := s.validateDashboardSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := firestore.Get().DeleteCommunityNote(session.Channel, req.ID); err != nil {
		log.Logger().Errorf(nil, "dashboard delete community note failed: %s", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "delete failed"})
		return
	}

	log.Logger().Infof(nil, "dashboard: deleted community note %s from %s", req.ID, session.Channel)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

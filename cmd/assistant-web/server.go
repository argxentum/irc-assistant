package main

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"assistant/pkg/queue"
	"context"
	"fmt"
	nativeLog "log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const dashboardRequestTimeout = 5 * time.Second

type server struct {
	ctx     context.Context
	cfg     *config.Config
	pending map[string]chan *models.DashboardResponseTaskData
	mu      sync.Mutex
}

var templatesRoot = "cmd/assistant-web/templates"

func (s *server) start() {
	logger := log.Logger()
	logger.Rawf(log.Info, "starting %s on :%d", s.cfg.Web.ExternalRootURL, s.cfg.Web.Port)

	// misc routes
	http.HandleFunc("/", s.defaultHandler)
	http.HandleFunc("/text/{text}", s.giphyAnimatedTextHandler)
	http.HandleFunc("/animated/{text}", s.giphyAnimatedTextHandler)
	http.HandleFunc("/gifs/{q}", s.giphySearchHandler)
	http.HandleFunc("/gif/{q}", s.giphySearchHandler)
	http.HandleFunc("/s/{id}", s.shortcutHandler)

	// task routes
	http.HandleFunc("POST /tasks/execute", s.taskExecuteHandler)

	// page routes
	http.HandleFunc("/about", s.aboutPageHandler)
	http.HandleFunc("/chat/{id}", s.llmSessionHandler)
	http.HandleFunc("/chat/{id}/poll", s.llmSessionPollHandler)

	// dashboard routes
	http.HandleFunc("/auth/{token}", s.dashboardAuthHandler)
	http.HandleFunc("/dashboard", s.dashboardHandler)
	http.HandleFunc("/dashboard/api/users", s.dashboardUsersHandler)
	http.HandleFunc("/dashboard/api/users/all", s.dashboardAllUsersHandler)
	http.HandleFunc("/dashboard/api/users/host/{host}", s.dashboardUsersByHostHandler)
	http.HandleFunc("/dashboard/api/users/mask/{mask...}", s.dashboardUsersByMaskHandler)
	http.HandleFunc("/dashboard/api/user/{nick}", s.dashboardUserHandler)
	http.HandleFunc("POST /dashboard/api/action/{action}", s.dashboardActionHandler)
	http.HandleFunc("/dashboard/api/voicerequests", s.dashboardVoiceRequestsHandler)
	http.HandleFunc("POST /dashboard/api/voicerequests/{action}", s.dashboardVoiceRequestActionHandler)
	http.HandleFunc("/dashboard/api/topic", s.dashboardGetTopicHandler)
	http.HandleFunc("POST /dashboard/api/topic", s.dashboardSetTopicHandler)
	http.HandleFunc("/dashboard/api/bans", s.dashboardBansHandler)
	http.HandleFunc("POST /dashboard/api/bans/add", s.dashboardAddBanHandler)
	http.HandleFunc("POST /dashboard/api/bans/remove", s.dashboardRemoveBanHandler)
	http.HandleFunc("/dashboard/api/stats", s.dashboardStatsHandler)
	http.HandleFunc("/dashboard/api/sources", s.dashboardSourcesHandler)
	http.HandleFunc("/dashboard/api/source/{id}", s.dashboardSourceHandler)
	http.HandleFunc("POST /dashboard/api/sources/save", s.dashboardSourceSaveHandler)
	http.HandleFunc("POST /dashboard/api/sources/delete", s.dashboardSourceDeleteHandler)
	http.HandleFunc("/dashboard/api/sources/top", s.dashboardTopSourcesHandler)
	http.HandleFunc("/dashboard/api/sources/unknown", s.dashboardUnknownSourcesHandler)
	http.HandleFunc("/dashboard/api/bannedwords", s.dashboardBannedWordsHandler)
	http.HandleFunc("POST /dashboard/api/bannedwords/add", s.dashboardBannedWordAddHandler)
	http.HandleFunc("POST /dashboard/api/bannedwords/remove", s.dashboardBannedWordRemoveHandler)
	http.HandleFunc("/dashboard/api/disinfo", s.dashboardDisinfoSourcesHandler)
	http.HandleFunc("POST /dashboard/api/disinfo/add", s.dashboardDisinfoSourceAddHandler)
	http.HandleFunc("POST /dashboard/api/disinfo/remove", s.dashboardDisinfoSourceRemoveHandler)
	http.HandleFunc("/dashboard/api/commands", s.dashboardCommandsHandler)
	http.HandleFunc("POST /dashboard/api/commands/toggle", s.dashboardCommandToggleHandler)
	http.HandleFunc("/dashboard/api/commands/usage", s.dashboardCommandUsageHandler)
	http.HandleFunc("/dashboard/api/penalties", s.dashboardPenaltiesHandler)
	http.HandleFunc("POST /dashboard/api/penalties/expire", s.dashboardExpirePenaltyHandler)

	nativeLog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Web.Port), nil))
}

func (s *server) receiveDashboardResponses() {
	logger := log.Logger()
	logger.Debug(nil, "listening for dashboard responses")

	err := queue.GetDashboardResponse().Receive(func(task *models.Task) {
		if task.Type != models.TaskTypeDashboardResponse {
			return
		}

		data := task.Data.(models.DashboardResponseTaskData)

		s.mu.Lock()
		ch, ok := s.pending[data.RequestID]
		if ok {
			delete(s.pending, data.RequestID)
		}
		s.mu.Unlock()

		if ok {
			ch <- &data
		}
	})

	if err != nil {
		logger.Errorf(nil, "error receiving dashboard responses: %s", err)
	}
}

func (s *server) dashboardRequest(data models.DashboardRequestTaskData) (*models.DashboardResponseTaskData, error) {
	requestID := uuid.NewString()
	ch := make(chan *models.DashboardResponseTaskData, 1)

	s.mu.Lock()
	s.pending[requestID] = ch
	s.mu.Unlock()

	task := models.NewDashboardRequestTask(requestID, data)
	if err := queue.GetDashboardRequest().Publish(task); err != nil {
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(dashboardRequestTimeout):
		s.mu.Lock()
		delete(s.pending, requestID)
		s.mu.Unlock()
		return nil, fmt.Errorf("request timed out")
	}
}

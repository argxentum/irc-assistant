package cloudtasks

import (
	"assistant/pkg/config"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"context"
	"fmt"
	"strings"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var instance *CloudTasks

type CloudTasks struct {
	ctx    context.Context
	cfg    *config.Config
	client *cloudtasks.Client
	queue  string
}

func Initialize(ctx context.Context, cfg *config.Config) (*CloudTasks, error) {
	if instance != nil {
		return instance, nil
	}

	client, err := cloudtasks.NewClient(ctx, option.WithCredentialsFile(cfg.GoogleCloud.ServiceAccountFilename))
	if err != nil {
		return nil, fmt.Errorf("error creating cloud tasks client: %w", err)
	}

	instance = &CloudTasks{
		ctx:    ctx,
		cfg:    cfg,
		client: client,
		queue:  fmt.Sprintf("projects/%s/locations/%s/queues/%s", cfg.GoogleCloud.ProjectID, cfg.CloudTasks.Location, cfg.CloudTasks.Queue),
	}

	return instance, nil
}

func Get() *CloudTasks {
	if instance == nil {
		panic("cloud tasks is not initialized")
	}
	return instance
}

func (ct *CloudTasks) Close() error {
	return ct.client.Close()
}

func (ct *CloudTasks) CreateTask(task *models.Task) (string, error) {
	logger := log.Logger()

	data, err := task.Serialize()
	if err != nil {
		return "", fmt.Errorf("error serializing task: %w", err)
	}

	handlerURL := ct.cfg.Web.ExternalRootURL + "/tasks/execute"

	// Cloud Tasks requires globally unique names and retains names for ~1 hour
	// after completion. Append a timestamp suffix to avoid collisions on
	// rescheduled tasks (e.g., persistent inactivity tasks reuse the same ID).
	taskID := task.ID
	if task.Type == models.TaskTypePersistentChannel {
		channel := task.Data.(models.PersistentTaskData).Channel
		taskID = fmt.Sprintf("%s-%s", taskID, strings.ReplaceAll(channel, "#", ""))
	}
	taskName := fmt.Sprintf("%s/tasks/%s-%d", ct.queue, taskID, task.DueAt.UnixMilli())

	req := &taskspb.CreateTaskRequest{
		Parent: ct.queue,
		Task: &taskspb.Task{
			Name:         taskName,
			ScheduleTime: timestamppb.New(task.DueAt),
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					Url:        handlerURL,
					HttpMethod: taskspb.HttpMethod_POST,
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer " + ct.cfg.CloudTasks.AuthToken,
					},
					Body: data,
				},
			},
		},
	}

	created, err := ct.client.CreateTask(ct.ctx, req)
	if err != nil {
		return "", fmt.Errorf("error creating cloud task %s: %w", task.ID, err)
	}

	logger.Debugf(nil, "created cloud task: %s", created.Name)
	return created.Name, nil
}

func (ct *CloudTasks) DeleteTask(taskID string) error {
	logger := log.Logger()

	err := ct.client.DeleteTask(ct.ctx, &taskspb.DeleteTaskRequest{
		Name: ct.taskName(taskID),
	})
	if err != nil {
		return fmt.Errorf("error deleting cloud task %s: %w", taskID, err)
	}

	logger.Debugf(nil, "deleted cloud task: %s", taskID)
	return nil
}

func (ct *CloudTasks) taskName(taskID string) string {
	return fmt.Sprintf("%s/tasks/%s", ct.queue, taskID)
}

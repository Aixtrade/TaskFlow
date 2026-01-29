package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	taskapp "github.com/Aixtrade/TaskFlow/internal/application/task"
	"github.com/Aixtrade/TaskFlow/internal/domain/task"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
)

type fakeClient struct {
	getInfoErr error
}

func (f *fakeClient) Enqueue(ctx context.Context, t *task.Task, opts ...asynqqueue.EnqueueOptions) (*asynq.TaskInfo, error) {
	return nil, nil
}

func (f *fakeClient) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	return nil, f.getInfoErr
}

func (f *fakeClient) ListTasks(queue, state string, page, size int) ([]*asynq.TaskInfo, error) {
	return nil, nil
}

func (f *fakeClient) CancelTask(taskID string) error {
	return nil
}

func (f *fakeClient) DeleteTask(queue, taskID string) error {
	return nil
}

func (f *fakeClient) GetQueueInfo(queue string) (*asynq.QueueInfo, error) {
	return nil, nil
}

func (f *fakeClient) GetAllQueueStats() ([]asynqqueue.QueueStats, error) {
	return nil, nil
}

func setupTaskRouter(service *taskapp.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewTaskHandler(service)
	r.POST("/api/v1/tasks", h.Create)
	r.GET("/api/v1/tasks/:id", h.Get)
	return r
}

func TestTaskHandlerGetNotFound(t *testing.T) {
	fake := &fakeClient{getInfoErr: asynq.ErrTaskNotFound}
	service := taskapp.NewService(fake, zap.NewNop())
	r := setupTaskRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/123?queue=default", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["code"] != "TASK_NOT_FOUND" {
		t.Fatalf("expected TASK_NOT_FOUND, got %s", body["code"])
	}
}

func TestTaskHandlerCreateInvalidRequest(t *testing.T) {
	service := taskapp.NewService(&fakeClient{}, zap.NewNop())
	r := setupTaskRouter(service)

	payload := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", payload)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["code"] != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %s", body["code"])
	}
}

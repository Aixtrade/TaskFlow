package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/domain/task"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	apperrors "github.com/Aixtrade/TaskFlow/pkg/errors"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type fakeClient struct {
	enqueueInfo *asynq.TaskInfo
	enqueueErr  error

	getInfo    *asynq.TaskInfo
	getInfoErr error

	cancelErr error
	deleteErr error

	queueInfo    *asynq.QueueInfo
	queueInfoErr error

	allStats    []asynqqueue.QueueStats
	allStatsErr error
}

func (f *fakeClient) Enqueue(ctx context.Context, t *task.Task, opts ...asynqqueue.EnqueueOptions) (*asynq.TaskInfo, error) {
	if f.enqueueErr != nil {
		return nil, f.enqueueErr
	}
	return f.enqueueInfo, nil
}

func (f *fakeClient) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	if f.getInfoErr != nil {
		return nil, f.getInfoErr
	}
	return f.getInfo, nil
}

func (f *fakeClient) ListTasks(queue, state string, page, size int) ([]*asynq.TaskInfo, error) {
	return nil, nil
}

func (f *fakeClient) CancelTask(taskID string) error {
	return f.cancelErr
}

func (f *fakeClient) DeleteTask(queue, taskID string) error {
	return f.deleteErr
}

func (f *fakeClient) GetQueueInfo(queue string) (*asynq.QueueInfo, error) {
	if f.queueInfoErr != nil {
		return nil, f.queueInfoErr
	}
	return f.queueInfo, nil
}

func (f *fakeClient) GetAllQueueStats() ([]asynqqueue.QueueStats, error) {
	if f.allStatsErr != nil {
		return nil, f.allStatsErr
	}
	return f.allStats, nil
}

func TestServiceCreateTaskAlreadyExists(t *testing.T) {
	fake := &fakeClient{enqueueErr: asynq.ErrTaskIDConflict}
	service := NewService(fake, zap.NewNop())

	cmd := &CreateTaskCommand{
		Type:    tasktype.Demo,
		Payload: []byte(`{"message":"hi","count":1}`),
	}

	_, err := service.CreateTask(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrTaskAlreadyExists) {
		t.Fatalf("expected ErrTaskAlreadyExists, got %v", err)
	}
}

func TestServiceGetTaskNotFound(t *testing.T) {
	fake := &fakeClient{getInfoErr: asynq.ErrTaskNotFound}
	service := NewService(fake, zap.NewNop())

	query := &GetTaskQuery{TaskID: "id", Queue: "default"}
	_, err := service.GetTask(context.Background(), query)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestServiceCancelTaskNotFound(t *testing.T) {
	fake := &fakeClient{cancelErr: asynq.ErrTaskNotFound}
	service := NewService(fake, zap.NewNop())

	err := service.CancelTask(context.Background(), &CancelTaskCommand{TaskID: "id"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestServiceDeleteTaskNotFound(t *testing.T) {
	fake := &fakeClient{deleteErr: asynq.ErrTaskNotFound}
	service := NewService(fake, zap.NewNop())

	err := service.DeleteTask(context.Background(), &DeleteTaskCommand{TaskID: "id", Queue: "default"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestServiceGetQueueStatsSingleQueue(t *testing.T) {
	fake := &fakeClient{
		queueInfo: &asynq.QueueInfo{
			Queue:     "default",
			Pending:   1,
			Active:    2,
			Scheduled: 3,
			Retry:     4,
			Archived:  5,
			Completed: 6,
		},
	}
	service := NewService(fake, zap.NewNop())

	stats, err := service.GetQueueStats(context.Background(), &GetQueueStatsQuery{Queue: "default"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Queue != "default" || stats[0].Pending != 1 || stats[0].Active != 2 || stats[0].Scheduled != 3 || stats[0].Retry != 4 || stats[0].Archived != 5 || stats[0].Completed != 6 {
		t.Fatalf("unexpected stats: %+v", stats[0])
	}
}

func TestServiceGetQueueStatsAll(t *testing.T) {
	fake := &fakeClient{
		allStats: []asynqqueue.QueueStats{{Queue: "default", Pending: 1}},
	}
	service := NewService(fake, zap.NewNop())

	stats, err := service.GetQueueStats(context.Background(), &GetQueueStatsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
}

func TestServiceCreateTaskUsesProcessAt(t *testing.T) {
	info := &asynq.TaskInfo{ID: "id", Queue: "default", State: asynq.TaskStatePending}
	fake := &fakeClient{enqueueInfo: info}
	service := NewService(fake, zap.NewNop())

	when := time.Now().Add(2 * time.Minute)
	cmd := &CreateTaskCommand{
		Type:      tasktype.Demo,
		Payload:   []byte(`{"message":"hi","count":1}`),
		ProcessAt: when,
	}

	result, err := service.CreateTask(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TaskID != "id" {
		t.Fatalf("expected task id 'id', got %s", result.TaskID)
	}
}

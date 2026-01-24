package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	taskapp "github.com/Aixtrade/TaskFlow/internal/application/task"
	"github.com/Aixtrade/TaskFlow/internal/interfaces/http/dto"
)

type TaskHandler struct {
	service *taskapp.Service
}

func NewTaskHandler(service *taskapp.Service) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_REQUEST",
		})
		return
	}

	timeout, err := req.GetTimeout()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid timeout format",
			Code:  "INVALID_TIMEOUT",
		})
		return
	}

	processAt, err := req.GetProcessAt()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid process_at format",
			Code:  "INVALID_PROCESS_AT",
		})
		return
	}

	unique, err := req.GetUnique()
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "invalid unique format",
			Code:  "INVALID_UNIQUE",
		})
		return
	}

	cmd := &taskapp.CreateTaskCommand{
		Type:       req.GetTaskType(),
		Payload:    req.Payload,
		Queue:      req.Queue,
		MaxRetries: req.MaxRetries,
		Timeout:    timeout,
		ProcessAt:  processAt,
		Unique:     unique,
		Metadata:   req.Metadata,
	}

	result, err := h.service.CreateTask(c.Request.Context(), cmd)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"

		switch err {
		case taskapp.ErrInvalidTaskType:
			status = http.StatusBadRequest
			code = "INVALID_TASK_TYPE"
		case taskapp.ErrInvalidPayload:
			status = http.StatusBadRequest
			code = "INVALID_PAYLOAD"
		}

		c.JSON(status, dto.ErrorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}

	c.JSON(http.StatusCreated, dto.CreateTaskResponse{
		TaskID: result.TaskID,
		Queue:  result.Queue,
		Status: result.Status,
	})
}

func (h *TaskHandler) Get(c *gin.Context) {
	taskID := c.Param("id")
	queue := c.Query("queue")

	if queue == "" {
		queue = "default"
	}

	query := &taskapp.GetTaskQuery{
		TaskID: taskID,
		Queue:  queue,
	}

	result, err := h.service.GetTask(c.Request.Context(), query)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"

		if err == taskapp.ErrTaskNotFound {
			status = http.StatusNotFound
			code = "TASK_NOT_FOUND"
		}

		c.JSON(status, dto.ErrorResponse{
			Error: err.Error(),
			Code:  code,
		})
		return
	}

	c.JSON(http.StatusOK, dto.GetTaskResponse{
		ID:            result.ID,
		Queue:         result.Queue,
		Type:          result.Type,
		State:         result.State,
		MaxRetry:      result.MaxRetry,
		Retried:       result.Retried,
		LastErr:       result.LastErr,
		NextProcessAt: result.NextProcessAt,
	})
}

func (h *TaskHandler) Cancel(c *gin.Context) {
	taskID := c.Param("id")

	cmd := &taskapp.CancelTaskCommand{
		TaskID: taskID,
	}

	err := h.service.CancelTask(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "CANCEL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task cancelled"})
}

func (h *TaskHandler) Delete(c *gin.Context) {
	taskID := c.Param("id")
	queue := c.Query("queue")

	if queue == "" {
		queue = "default"
	}

	cmd := &taskapp.DeleteTaskCommand{
		TaskID: taskID,
		Queue:  queue,
	}

	err := h.service.DeleteTask(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "DELETE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

func (h *TaskHandler) GetQueueStats(c *gin.Context) {
	queue := c.Query("queue")

	query := &taskapp.GetQueueStatsQuery{
		Queue: queue,
	}

	stats, err := h.service.GetQueueStats(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "STATS_FAILED",
		})
		return
	}

	response := make([]dto.QueueStatsResponse, len(stats))
	for i, s := range stats {
		response[i] = dto.QueueStatsResponse{
			Queue:     s.Queue,
			Pending:   s.Pending,
			Active:    s.Active,
			Scheduled: s.Scheduled,
			Retry:     s.Retry,
			Archived:  s.Archived,
			Completed: s.Completed,
		}
	}

	c.JSON(http.StatusOK, response)
}

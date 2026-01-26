package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/pkg/progress"
)

// ProgressHandler 处理进度相关的 HTTP 请求
type ProgressHandler struct {
	subscriber *progress.Subscriber
	logger     *zap.Logger
}

// NewProgressHandler 创建进度处理器
func NewProgressHandler(subscriber *progress.Subscriber, logger *zap.Logger) *ProgressHandler {
	return &ProgressHandler{
		subscriber: subscriber,
		logger:     logger,
	}
}

// StreamProgress 通过 SSE 流式推送任务进度
// GET /api/v1/tasks/:id/progress/stream
func (h *ProgressHandler) StreamProgress(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	// 可选参数：从指定位置开始读取
	// - "0" 或 "0-0": 从头开始读取（包含历史）
	// - "$" 或空: 只读取新消息
	// - 具体 ID: 从该 ID 之后开始读取
	startID := c.Query("start_id")
	if startID == "" {
		startID = "$" // 默认只读取新消息
	}

	// 可选参数：是否包含历史进度
	includeHistory := c.Query("history") == "true"

	h.logger.Info("SSE connection established",
		zap.String("task_id", taskID),
		zap.String("start_id", startID),
		zap.Bool("include_history", includeHistory),
	)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 如果请求历史进度，先发送历史数据
	if includeHistory {
		h.sendHistory(c, taskID)
	}

	ctx := c.Request.Context()

	// 订阅进度更新
	ch := h.subscriber.Subscribe(ctx, taskID, startID)

	c.Stream(func(w io.Writer) bool {
		select {
		case result, ok := <-ch:
			if !ok {
				// channel 已关闭
				return false
			}

			if result.Error != nil {
				// 发送错误事件
				h.writeSSEEvent(w, "error", map[string]string{
					"message": result.Error.Error(),
				})
				return false
			}

			if result.IsFinal {
				// 发送最终进度
				h.writeSSEEvent(w, "progress", result.Progress)
				// 发送完成事件
				h.writeSSEEvent(w, "done", map[string]interface{}{
					"task_id": taskID,
					"status":  result.Status,
				})
				return false
			}

			// 发送进度事件
			h.writeSSEEvent(w, "progress", result.Progress)
			return true

		case <-ctx.Done():
			h.logger.Debug("SSE connection closed by client",
				zap.String("task_id", taskID),
			)
			return false
		}
	})
}

// sendHistory 发送历史进度
func (h *ProgressHandler) sendHistory(c *gin.Context, taskID string) {
	history, err := h.subscriber.GetHistory(c.Request.Context(), taskID, "-", 0)
	if err != nil {
		h.logger.Warn("failed to get history",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return
	}

	for _, result := range history {
		if result.Progress != nil {
			h.writeSSEEvent(c.Writer, "history", result.Progress)
		}
	}
}

// writeSSEEvent 写入 SSE 事件
func (h *ProgressHandler) writeSSEEvent(w io.Writer, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("failed to marshal SSE data", zap.Error(err))
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)

	// 刷新缓冲区
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// GetLatestProgress 获取最新进度（非流式）
// GET /api/v1/tasks/:id/progress
func (h *ProgressHandler) GetLatestProgress(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	result, err := h.subscriber.GetLatest(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get progress",
			"code":  "PROGRESS_FETCH_ERROR",
		})
		return
	}

	if result == nil || result.Progress == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no progress found for this task",
			"code":  "PROGRESS_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"progress":  result.Progress,
		"is_final":  result.IsFinal,
		"status":    result.Status,
		"stream_id": result.StreamID,
	})
}

// GetProgressHistory 获取进度历史
// GET /api/v1/tasks/:id/progress/history
func (h *ProgressHandler) GetProgressHistory(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	// 可选参数
	startID := c.DefaultQuery("start_id", "-")
	count := int64(100) // 默认返回最近 100 条

	history, err := h.subscriber.GetHistory(c.Request.Context(), taskID, startID, count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get progress history",
			"code":  "PROGRESS_HISTORY_ERROR",
		})
		return
	}

	// 转换为响应格式
	items := make([]gin.H, 0, len(history))
	for _, result := range history {
		item := gin.H{
			"stream_id": result.StreamID,
			"progress":  result.Progress,
			"is_final":  result.IsFinal,
		}
		if result.IsFinal {
			item["status"] = result.Status
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"count":   len(items),
		"history": items,
	})
}

// GetProgressInfo 获取进度 Stream 信息
// GET /api/v1/tasks/:id/progress/info
func (h *ProgressHandler) GetProgressInfo(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	info, err := h.subscriber.GetStreamInfo(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get stream info",
			"code":  "STREAM_INFO_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":      taskID,
		"has_progress": info.HasProgress,
		"length":       info.Length,
		"first_entry":  info.FirstEntry,
		"last_entry":   info.LastEntry,
	})
}

// StreamMultipleProgress 同时订阅多个任务的进度
// GET /api/v1/progress/stream?task_ids=id1,id2,id3
func (h *ProgressHandler) StreamMultipleProgress(c *gin.Context) {
	taskIDsParam := c.Query("task_ids")
	if taskIDsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_ids is required"})
		return
	}

	taskIDs := strings.Split(taskIDsParam, ",")
	if len(taskIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one task_id is required"})
		return
	}

	if len(taskIDs) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 10 tasks can be subscribed at once"})
		return
	}

	h.logger.Info("SSE multi-task connection established",
		zap.Strings("task_ids", taskIDs),
	)

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()

	// 为每个任务创建订阅
	type taggedResult struct {
		TaskID string
		Result progress.SubscribeResult
	}

	merged := make(chan taggedResult, len(taskIDs)*10)

	// 启动订阅
	for _, taskID := range taskIDs {
		taskID := taskID // 捕获变量
		ch := h.subscriber.Subscribe(ctx, taskID, "$")

		go func() {
			for result := range ch {
				select {
				case merged <- taggedResult{TaskID: taskID, Result: result}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	activeTasks := len(taskIDs)

	c.Stream(func(w io.Writer) bool {
		select {
		case tr := <-merged:
			result := tr.Result

			if result.Error != nil {
				h.writeSSEEvent(w, "error", map[string]string{
					"task_id": tr.TaskID,
					"message": result.Error.Error(),
				})
				activeTasks--
				return activeTasks > 0
			}

			// 发送带有 task_id 的进度
			eventData := map[string]interface{}{
				"task_id":  tr.TaskID,
				"progress": result.Progress,
			}

			if result.IsFinal {
				eventData["is_final"] = true
				eventData["status"] = result.Status
				h.writeSSEEvent(w, "progress", eventData)
				activeTasks--
				return activeTasks > 0
			}

			h.writeSSEEvent(w, "progress", eventData)
			return true

		case <-ctx.Done():
			return false
		}
	})
}

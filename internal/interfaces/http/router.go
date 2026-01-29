package http

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	taskapp "github.com/Aixtrade/TaskFlow/internal/application/task"
	"github.com/Aixtrade/TaskFlow/internal/config"
	"github.com/Aixtrade/TaskFlow/internal/interfaces/http/handler"
	"github.com/Aixtrade/TaskFlow/internal/interfaces/http/middleware"
	"github.com/Aixtrade/TaskFlow/pkg/progress"
)

type Router struct {
	engine             *gin.Engine
	cfg                *config.Config
	logger             *zap.Logger
	taskService        *taskapp.Service
	redisClient        *redis.Client
	progressSubscriber *progress.Subscriber
}

type RouterConfig struct {
	Config      *config.Config
	Logger      *zap.Logger
	TaskService *taskapp.Service
	RedisClient *redis.Client
	Progress    progress.StreamOptions
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 创建进度订阅器
	progressSubscriber := progress.NewSubscriber(cfg.RedisClient, cfg.Logger, cfg.Progress)

	return &Router{
		engine:             engine,
		cfg:                cfg.Config,
		logger:             cfg.Logger,
		taskService:        cfg.TaskService,
		redisClient:        cfg.RedisClient,
		progressSubscriber: progressSubscriber,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.engine.Use(middleware.Recovery(r.logger))
	r.engine.Use(middleware.RequestID())
	r.engine.Use(middleware.Logger(r.logger))
	r.engine.Use(middleware.CORS())

	r.setupHealthRoutes()
	r.setupAPIRoutes()

	return r.engine
}

func (r *Router) setupHealthRoutes() {
	healthHandler := handler.NewHealthHandler(r.redisClient)

	r.engine.GET("/health", healthHandler.Health)
	r.engine.GET("/ready", healthHandler.Ready)
	r.engine.GET("/live", healthHandler.Live)
}

func (r *Router) setupAPIRoutes() {
	taskHandler := handler.NewTaskHandler(r.taskService)
	progressHandler := handler.NewProgressHandler(r.progressSubscriber, r.logger)

	v1 := r.engine.Group("/api/v1")
	{
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("/:id", taskHandler.Get)
			tasks.DELETE("/:id", taskHandler.Delete)
			tasks.POST("/:id/cancel", taskHandler.Cancel)

			// 进度相关端点
			tasks.GET("/:id/progress", progressHandler.GetLatestProgress)
			tasks.GET("/:id/progress/stream", progressHandler.StreamProgress)
			tasks.GET("/:id/progress/history", progressHandler.GetProgressHistory)
			tasks.GET("/:id/progress/info", progressHandler.GetProgressInfo)
		}

		queues := v1.Group("/queues")
		{
			queues.GET("/stats", taskHandler.GetQueueStats)
		}

		// 批量进度订阅
		progress := v1.Group("/progress")
		{
			progress.GET("/stream", progressHandler.StreamMultipleProgress)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

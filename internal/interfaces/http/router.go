package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	taskapp "github.com/Aixtrade/TaskFlow/internal/application/task"
	"github.com/Aixtrade/TaskFlow/internal/config"
	"github.com/Aixtrade/TaskFlow/internal/interfaces/http/handler"
	"github.com/Aixtrade/TaskFlow/internal/interfaces/http/middleware"
)

type Router struct {
	engine      *gin.Engine
	cfg         *config.Config
	logger      *zap.Logger
	taskService *taskapp.Service
	redisClient *redis.Client
}

type RouterConfig struct {
	Config      *config.Config
	Logger      *zap.Logger
	TaskService *taskapp.Service
	RedisClient *redis.Client
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	return &Router{
		engine:      engine,
		cfg:         cfg.Config,
		logger:      cfg.Logger,
		taskService: cfg.TaskService,
		redisClient: cfg.RedisClient,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.engine.Use(middleware.Recovery(r.logger))
	r.engine.Use(middleware.Logger(r.logger))
	r.engine.Use(middleware.Metrics())
	r.engine.Use(middleware.CORS())
	r.engine.Use(middleware.RequestID())

	r.setupHealthRoutes()
	r.setupMetricsRoutes()
	r.setupAPIRoutes()

	return r.engine
}

func (r *Router) setupHealthRoutes() {
	healthHandler := handler.NewHealthHandler(r.redisClient)

	r.engine.GET("/health", healthHandler.Health)
	r.engine.GET("/ready", healthHandler.Ready)
	r.engine.GET("/live", healthHandler.Live)
}

func (r *Router) setupMetricsRoutes() {
	if r.cfg.Metrics.Enabled {
		r.engine.GET(r.cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}
}

func (r *Router) setupAPIRoutes() {
	taskHandler := handler.NewTaskHandler(r.taskService)

	v1 := r.engine.Group("/api/v1")
	{
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", taskHandler.Create)
			tasks.GET("/:id", taskHandler.Get)
			tasks.DELETE("/:id", taskHandler.Delete)
			tasks.POST("/:id/cancel", taskHandler.Cancel)
		}

		queues := v1.Group("/queues")
		{
			queues.GET("/stats", taskHandler.GetQueueStats)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

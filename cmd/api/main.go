package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	taskapp "github.com/Aixtrade/TaskFlow/internal/application/task"
	"github.com/Aixtrade/TaskFlow/internal/config"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	"github.com/Aixtrade/TaskFlow/internal/infrastructure/observability/logging"
	httpserver "github.com/Aixtrade/TaskFlow/internal/interfaces/http"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := logging.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("starting taskflow api",
		zap.String("env", cfg.App.Env),
		zap.String("host", cfg.Server.HTTP.Host),
		zap.Int("port", cfg.Server.HTTP.Port),
	)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}

	asynqClient, err := asynqqueue.NewClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("failed to create asynq client", zap.Error(err))
	}
	defer asynqClient.Close()

	taskService := taskapp.NewService(asynqClient, logger)

	router := httpserver.NewRouter(httpserver.RouterConfig{
		Config:      cfg,
		Logger:      logger,
		TaskService: taskService,
		RedisClient: redisClient,
	})

	engine := router.Setup()

	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting http server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

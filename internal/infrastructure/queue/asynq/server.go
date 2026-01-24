package asynq

import (
	"context"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/config"
)

type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	logger *zap.Logger
}

type ServerConfig struct {
	Redis       *config.RedisConfig
	Queues      map[string]int
	Concurrency int
	Logger      *zap.Logger
}

func NewServer(cfg ServerConfig) (*Server, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Concurrency,
			Queues:      cfg.Queues,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				cfg.Logger.Error("task error",
					zap.String("type", task.Type()),
					zap.Error(err),
				)
			}),
			Logger: newZapLogger(cfg.Logger),
		},
	)

	return &Server{
		server: server,
		mux:    asynq.NewServeMux(),
		logger: cfg.Logger,
	}, nil
}

func (s *Server) HandleFunc(pattern string, handler func(context.Context, *asynq.Task) error) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Handle(pattern string, handler asynq.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) Use(middlewares ...asynq.MiddlewareFunc) {
	s.mux.Use(middlewares...)
}

func (s *Server) Start() error {
	s.logger.Info("starting asynq server")
	return s.server.Start(s.mux)
}

func (s *Server) Shutdown() {
	s.logger.Info("shutting down asynq server")
	s.server.Shutdown()
}

func (s *Server) Stop() {
	s.logger.Info("stopping asynq server")
	s.server.Stop()
}

type zapLogger struct {
	logger *zap.Logger
}

func newZapLogger(l *zap.Logger) *zapLogger {
	return &zapLogger{logger: l.Named("asynq")}
}

func (l *zapLogger) Debug(args ...interface{}) {
	l.logger.Sugar().Debug(args...)
}

func (l *zapLogger) Info(args ...interface{}) {
	l.logger.Sugar().Info(args...)
}

func (l *zapLogger) Warn(args ...interface{}) {
	l.logger.Sugar().Warn(args...)
}

func (l *zapLogger) Error(args ...interface{}) {
	l.logger.Sugar().Error(args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.logger.Sugar().Fatal(args...)
}

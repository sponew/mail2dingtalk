package main

import (
	"fmt"
	"log/slog"
	"mail2dingtalk/config"
	"mail2dingtalk/dingtalk"
	"mail2dingtalk/parser"
	"mail2dingtalk/smtp"
	"mail2dingtalk/storage"
	"os"
	"os/signal"
	"syscall"
)

type EmailProcessor struct {
	dingtalkClient *dingtalk.Client
	emailStorage   *storage.EmailStorage
	logger         *slog.Logger
}

func (p *EmailProcessor) Process(email *parser.ParsedEmail) error {
	p.logger.Info("processing email", "id", email.ID, "subject", email.Subject)

	if err := p.dingtalkClient.SendEmailNotification(email); err != nil {
		p.logger.Error("send to dingtalk", "error", err)
	}

	if err := p.emailStorage.Save(email); err != nil {
		p.logger.Error("save email", "error", err)
		return fmt.Errorf("save email: %w", err)
	}

	p.logger.Info("email processed successfully", "id", email.ID)
	return nil
}

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg)

	dingtalkClient := dingtalk.NewClient()
	emailStorage := storage.NewEmailStorage()

	processor := &EmailProcessor{
		dingtalkClient: dingtalkClient,
		emailStorage:   emailStorage,
		logger:         logger,
	}

	emailQueue := smtp.NewEmailQueue(cfg.Server.MaxConcurrent, processor, logger)
	emailQueue.Start(cfg.Server.MaxConcurrent)

	smtpServer := smtp.NewServer(cfg, &QueueWrapper{queue: emailQueue, logger: logger}, logger)

	storage.StartCleanupScheduler(cfg.Storage.RetentionDays)
	logger.Info("started cleanup scheduler", "retention_days", cfg.Storage.RetentionDays)

	go func() {
		if err := smtpServer.ListenAndServe(); err != nil {
			logger.Error("smtp server error", "error", err)
		}
	}()

	logger.Info("mail2dingtalk service started")
	logger.Info("SMTP listening on port", "port", cfg.SMTP.Port)
	logger.Info("Max concurrent workers", "count", cfg.Server.MaxConcurrent)
	logger.Info("Attachment max size", "size_mb", cfg.Attachment.MaxSizeMB)
	logger.Info("Email retention", "days", cfg.Storage.RetentionDays)

	waitForShutdown(smtpServer, emailQueue)
}

type QueueWrapper struct {
	queue  *smtp.EmailQueue
	logger *slog.Logger
}

func (w *QueueWrapper) Process(email *parser.ParsedEmail) error {
	w.queue.Add(email)
	return nil
}

func setupLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	
	if cfg.Log.File != "" {
		if err := os.MkdirAll(filepathDir(cfg.Log.File), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log dir: %v\n", err)
		}
		
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		} else {
			handler = slog.NewTextHandler(file, &slog.HandlerOptions{Level: level})
		}
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func waitForShutdown(server *smtp.Server, queue *smtp.EmailQueue) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nShutting down...")

	if err := server.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "Error shutting down SMTP server: %v\n", err)
	}

	queue.Shutdown()
	fmt.Println("Gracefully stopped")
}

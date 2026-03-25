package smtp

import (
	"fmt"
	"io"
	"log/slog"
	"mail2dingtalk/config"
	"mail2dingtalk/parser"
	"sync"

	"github.com/emersion/go-smtp"
)

type EmailProcessor interface {
	Process(email *parser.ParsedEmail) error
}

type Backend struct {
	processor EmailProcessor
	logger    *slog.Logger
}

func NewBackend(processor EmailProcessor, logger *slog.Logger) *Backend {
	return &Backend{
		processor: processor,
		logger:    logger,
	}
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		backend: b,
		logger:  b.logger,
	}, nil
}

type Session struct {
	backend *Backend
	logger  *slog.Logger
	from    string
	to      []string
	data    []byte
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	s.logger.Debug("mail from", "from", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	s.logger.Debug("rcpt to", "to", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		s.logger.Error("read data", "error", err)
		return err
	}
	s.data = data
	s.logger.Info("received email", "from", s.from, "to", s.to, "size", len(data))

	parsed, err := parser.ParseEmail(data)
	if err != nil {
		s.logger.Error("parse email", "error", err)
		return err
	}

	if err := s.backend.processor.Process(parsed); err != nil {
		s.logger.Error("process email", "error", err)
		return err
	}

	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

type Server struct {
	server *smtp.Server
	logger *slog.Logger
}

func NewServer(cfg *config.Config, processor EmailProcessor, logger *slog.Logger) *Server {
	backend := NewBackend(processor, logger)
	s := smtp.NewServer(backend)
	s.Addr = ":" + fmt.Sprintf("%d", cfg.SMTP.Port)
	s.Domain = cfg.SMTP.Domain
	s.AllowInsecureAuth = true
	return &Server{server: s, logger: logger}
}

func (s *Server) ListenAndServe() error {
	s.logger.Info("starting SMTP server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	s.logger.Info("shutting down SMTP server")
	return s.server.Close()
}

type EmailQueue struct {
	queue    chan *parser.ParsedEmail
	processor EmailProcessor
	logger   *slog.Logger
	wg       sync.WaitGroup
}

func NewEmailQueue(size int, processor EmailProcessor, logger *slog.Logger) *EmailQueue {
	return &EmailQueue{
		queue:    make(chan *parser.ParsedEmail, size),
		processor: processor,
		logger:   logger,
	}
}

func (q *EmailQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	q.logger.Info("started email workers", "count", workers)
}

func (q *EmailQueue) worker() {
	defer q.wg.Done()
	for email := range q.queue {
		if err := q.processor.Process(email); err != nil {
			q.logger.Error("worker process email", "error", err)
		}
	}
}

func (q *EmailQueue) Add(email *parser.ParsedEmail) {
	q.queue <- email
}

func (q *EmailQueue) Shutdown() {
	close(q.queue)
	q.wg.Wait()
}

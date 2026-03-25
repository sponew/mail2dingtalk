package storage

import (
	"encoding/json"
	"fmt"
	"mail2dingtalk/config"
	"mail2dingtalk/parser"
	"os"
	"path/filepath"
	"time"
)

type EmailStorage struct {
	emailDir string
}

type EmailRecord struct {
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	ReceivedAt  time.Time `json:"received_at"`
	BodyText    string    `json:"body_text,omitempty"`
	BodyHTML    string    `json:"body_html,omitempty"`
	Attachments []AttachmentInfo `json:"attachments,omitempty"`
}

type AttachmentInfo struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	Path      string `json:"path"`
	TooLarge  bool   `json:"too_large"`
}

func NewEmailStorage() *EmailStorage {
	cfg := config.Global
	return &EmailStorage{
		emailDir: cfg.Storage.EmailDir,
	}
}

func (s *EmailStorage) Save(email *parser.ParsedEmail) error {
	if err := os.MkdirAll(s.emailDir, 0755); err != nil {
		return fmt.Errorf("create email dir: %w", err)
	}

	record := EmailRecord{
		ID:         email.ID,
		From:       email.From,
		To:         email.To,
		Subject:    email.Subject,
		Date:       email.Date,
		ReceivedAt: time.Now(),
		BodyText:   email.BodyText,
		BodyHTML:   email.BodyHTML,
	}

	for _, att := range email.Attachments {
		record.Attachments = append(record.Attachments, AttachmentInfo{
			Filename: att.Filename,
			Size:     att.Size,
			Path:     att.Path,
			TooLarge: att.TooLarge,
		})
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal email: %w", err)
	}

	filename := fmt.Sprintf("%s.json", email.ID)
	path := filepath.Join(s.emailDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write email file: %w", err)
	}

	return nil
}

func (s *EmailStorage) CleanOldEmails(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	entries, err := os.ReadDir(s.emailDir)
	if err != nil {
		return fmt.Errorf("read email dir: %w", err)
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.emailDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := s.deleteEmail(path); err != nil {
				continue
			}
			deletedCount++
		}
	}

	return nil
}

func (s *EmailStorage) deleteEmail(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var record EmailRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return err
	}

	for _, att := range record.Attachments {
		if att.Path != "" {
			os.Remove(att.Path)
		}
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}

func StartCleanupScheduler(retentionDays int) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		storage := NewEmailStorage()
		if err := storage.CleanOldEmails(retentionDays); err != nil {
			fmt.Printf("initial cleanup error: %v\n", err)
		}

		for range ticker.C {
			if err := storage.CleanOldEmails(retentionDays); err != nil {
				fmt.Printf("cleanup error: %v\n", err)
			}
		}
	}()
}

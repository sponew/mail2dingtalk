package parser

import (
	"fmt"
	"io"
	"mail2dingtalk/config"
	"os"
	"path/filepath"
	"strings"
	"time"

	markdown "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/emersion/go-message/mail"
)

type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Path        string
	TooLarge    bool
}

type ParsedEmail struct {
	ID           string
	From         string
	To           []string
	Subject      string
	Date         time.Time
	BodyText     string
	BodyHTML     string
	BodyMarkdown string
	Attachments  []Attachment
	RawData      []byte
}

func ParseEmail(rawData []byte) (*ParsedEmail, error) {
	msg, err := mail.CreateReader(strings.NewReader(string(rawData)))
	if err != nil {
		return nil, fmt.Errorf("create mail reader: %w", err)
	}
	defer msg.Close()

	email := &ParsedEmail{
		ID:      generateID(),
		RawData: rawData,
	}

	if subject, err := msg.Header.Subject(); err == nil {
		email.Subject = subject
	}

	if addrs, err := msg.Header.AddressList("From"); err == nil && len(addrs) > 0 {
		email.From = addrs[0].String()
	}

	if addrs, err := msg.Header.AddressList("To"); err == nil && len(addrs) > 0 {
		for _, a := range addrs {
			email.To = append(email.To, a.String())
		}
	}

	if date, err := msg.Header.Date(); err == nil {
		email.Date = date
	} else {
		email.Date = time.Now()
	}

	for {
		part, err := msg.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read part: %w", err)
		}

		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			mediaType, _, _ := h.ContentType()
			body, _ := io.ReadAll(part.Body)
			switch mediaType {
			case "text/plain":
				if email.BodyText == "" {
					email.BodyText = string(body)
					email.BodyMarkdown = email.BodyText
				}
			case "text/html":
				if email.BodyHTML == "" {
					email.BodyHTML = string(body)
					email.BodyMarkdown = htmlToMarkdown(email.BodyHTML)
				}
			}
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			if filename == "" {
				filename = fmt.Sprintf("attachment_%s", generateID())
			}

			body, _ := io.ReadAll(part.Body)
			size := int64(len(body))
			mediaType, _, _ := h.ContentType()

			cfg := config.Global
			maxSize := int64(cfg.Attachment.MaxSizeMB * 1024 * 1024)
			tooLarge := size > maxSize

			attachment := Attachment{
				Filename:    filename,
				ContentType: mediaType,
				Size:        size,
				TooLarge:    tooLarge,
			}

			if !tooLarge {
				path := filepath.Join(cfg.Storage.AttachmentDir, fmt.Sprintf("%s_%s", email.ID, filename))
				if err := os.MkdirAll(cfg.Storage.AttachmentDir, 0755); err != nil {
					return nil, fmt.Errorf("create attachment dir: %w", err)
				}
				if err := os.WriteFile(path, body, 0644); err != nil {
					return nil, fmt.Errorf("write attachment: %w", err)
				}
				attachment.Path = path
			}

			email.Attachments = append(email.Attachments, attachment)
		}
	}

	if email.BodyMarkdown == "" {
		if email.BodyText != "" {
			email.BodyMarkdown = email.BodyText
		} else if email.BodyHTML != "" {
			email.BodyMarkdown = htmlToMarkdown(email.BodyHTML)
		}
	}

	return email, nil
}

func htmlToMarkdown(html string) string {
	converter := markdown.NewConverter("", true, nil)
	md, err := converter.ConvertString(html)
	if err != nil {
		return html
	}
	return strings.TrimSpace(md)
}

func generateID() string {
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

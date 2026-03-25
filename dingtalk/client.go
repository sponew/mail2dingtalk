package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mail2dingtalk/config"
	"mail2dingtalk/parser"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	webhookURL string
	secret     string
	client     *http.Client
}

type Message struct {
	Msgtype  string          `json:"msgtype"`
	Markdown *MarkdownContent `json:"markdown,omitempty"`
	Text     *TextContent    `json:"text,omitempty"`
	File     *FileContent    `json:"file,omitempty"`
}

type MarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type TextContent struct {
	Content string `json:"content"`
}

type FileContent struct {
	FileID string `json:"file_id"`
}

type UploadResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	FileID  string `json:"file_id"`
}

type SendResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func NewClient() *Client {
	cfg := config.Global
	return &Client{
		webhookURL: cfg.DingTalk.WebhookURL,
		secret:     cfg.DingTalk.Secret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SendEmailNotification(email *parser.ParsedEmail) error {
	markdown := c.buildEmailMarkdown(email)

	msg := Message{
		Msgtype: "markdown",
		Markdown: &MarkdownContent{
			Title: email.Subject,
			Text:  markdown,
		},
	}

	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("send markdown message: %w", err)
	}

	for _, attachment := range email.Attachments {
		if attachment.TooLarge {
			if err := c.sendTooLargeAttachmentNotice(attachment); err != nil {
				return fmt.Errorf("send too large notice: %w", err)
			}
			continue
		}

		if err := c.sendFile(attachment.Path, attachment.Filename); err != nil {
			return fmt.Errorf("send file %s: %w", attachment.Filename, err)
		}
	}

	return nil
}

func (c *Client) buildEmailMarkdown(email *parser.ParsedEmail) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 📧 新邮件通知\n\n"))
	sb.WriteString(fmt.Sprintf("**发件人**: %s\n\n", email.From))
	sb.WriteString(fmt.Sprintf("**收件人**: %s\n\n", strings.Join(email.To, "; ")))
	sb.WriteString(fmt.Sprintf("**主题**: %s\n\n", email.Subject))
	sb.WriteString(fmt.Sprintf("**时间**: %s\n\n", email.Date.Format("2006-01-02 15:04:05")))
	sb.WriteString("---\n\n")

	if email.BodyMarkdown != "" {
		sb.WriteString("### 邮件内容\n\n")
		sb.WriteString(email.BodyMarkdown)
		sb.WriteString("\n\n")
	}

	if len(email.Attachments) > 0 {
		sb.WriteString("---\n\n")
		sb.WriteString(fmt.Sprintf("### 附件 (%d 个)\n\n", len(email.Attachments)))
		for i, att := range email.Attachments {
			if att.TooLarge {
				sb.WriteString(fmt.Sprintf("%d. 📎 %s (**过大** - %.2f MB)\n", i+1, att.Filename, float64(att.Size)/1024/1024))
			} else {
				sb.WriteString(fmt.Sprintf("%d. 📎 %s (%.2f KB)\n", i+1, att.Filename, float64(att.Size)/1024))
			}
		}
	}

	return sb.String()
}

func (c *Client) sendTooLargeAttachmentNotice(attachment parser.Attachment) error {
	content := fmt.Sprintf("附件过大通知\n\n文件名: %s\n大小: %.2f MB (限制 20MB)\n已保存到: %s\n已跳过发送",
		attachment.Filename,
		float64(attachment.Size)/1024/1024,
		attachment.Path)

	msg := Message{
		Msgtype: "text",
		Text:    &TextContent{Content: content},
	}

	return c.sendMessage(msg)
}

func (c *Client) sendFile(filePath, filename string) error {
	file, err := osOpen(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() > 20*1024*1024 {
		return fmt.Errorf("file too large: %d bytes", fileInfo.Size())
	}

	content := fmt.Sprintf("📎 发送附件：%s\n大小：%.2f KB\n文件已保存至服务器：%s",
		filename,
		float64(fileInfo.Size())/1024,
		filePath)

	msg := Message{
		Msgtype: "text",
		Text:    &TextContent{Content: content},
	}

	return c.sendMessage(msg)
}

func (c *Client) sendMessage(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	reqURL := c.webhookURL
	if c.secret != "" {
		timestamp := time.Now().UnixMilli()
		signature := generateSignature(c.secret, timestamp)
		reqURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", c.webhookURL, timestamp, signature)
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var sendResp SendResponse
	if err := json.Unmarshal(respBody, &sendResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if sendResp.ErrCode != 0 {
		return fmt.Errorf("dingtalk error: %d - %s", sendResp.ErrCode, sendResp.ErrMsg)
	}

	return nil
}

func (c *Client) addSignature(req *http.Request) {
	if c.secret != "" {
		timestamp := time.Now().UnixMilli()
		signature := generateSignature(c.secret, timestamp)
		q := req.URL.Query()
		q.Set("timestamp", fmt.Sprintf("%d", timestamp))
		q.Set("sign", signature)
		req.URL.RawQuery = q.Encode()
	}
}

func generateSignature(secret string, timestamp int64) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}

var osOpen = os.Open

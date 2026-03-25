package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SMTP        SMTPConfig        `yaml:"smtp"`
	DingTalk    DingTalkConfig    `yaml:"dingtalk"`
	Storage     StorageConfig     `yaml:"storage"`
	Server      ServerConfig      `yaml:"server"`
	Attachment  AttachmentConfig  `yaml:"attachment"`
	Log         LogConfig         `yaml:"log"`
}

type SMTPConfig struct {
	Port   int    `yaml:"port"`
	Domain string `yaml:"domain"`
}

type DingTalkConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Secret     string `yaml:"secret"`
}

type StorageConfig struct {
	EmailDir       string `yaml:"email_dir"`
	AttachmentDir  string `yaml:"attachment_dir"`
	RetentionDays  int    `yaml:"retention_days"`
}

type ServerConfig struct {
	MaxConcurrent int `yaml:"max_concurrent"`
}

type AttachmentConfig struct {
	MaxSizeMB int `yaml:"max_size_mb"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

var Global *Config

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.setDefaults()
	Global = &cfg
	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.SMTP.Port == 0 {
		c.SMTP.Port = 2525
	}
	if c.SMTP.Domain == "" {
		c.SMTP.Domain = "localhost"
	}
	if c.Server.MaxConcurrent == 0 {
		c.Server.MaxConcurrent = 10
	}
	if c.Attachment.MaxSizeMB == 0 {
		c.Attachment.MaxSizeMB = 20
	}
	if c.Storage.RetentionDays == 0 {
		c.Storage.RetentionDays = 180
	}
	if c.Storage.EmailDir == "" {
		c.Storage.EmailDir = "data/emails"
	}
	if c.Storage.AttachmentDir == "" {
		c.Storage.AttachmentDir = "tmp/attachments"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.File == "" {
		c.Log.File = "logs/app.log"
	}
}

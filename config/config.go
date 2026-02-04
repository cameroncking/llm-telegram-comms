package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	TelegramToken          string            `json:"telegram_token"`
	UserAllowlistRequired  bool              `json:"user_allowlist_required"`
	GroupAllowlistRequired bool              `json:"group_allowlist_required"`
	UserAllowlist          []int64           `json:"user_allowlist"`
	GroupAllowlist         []int64           `json:"group_allowlist"`
	BackendCommand         string            `json:"backend_command"`
	WorkingDirectory       string            `json:"working_directory"`
	Environment            map[string]string `json:"environment"`
	DropEnvironment        bool              `json:"drop_environment"`
	EnableAttachments        bool              `json:"enable_attachments"`
	AttachmentPath           string            `json:"attachment_path"`
	AttachmentMethod         string            `json:"attachment_method"`
	AttachmentPathChatPrefix string            `json:"attachment_path_chat_prefix"`
	AttachmentPathChatIDSuffix bool              `json:"attachment_path_chatid_suffix"`
	AggressiveShellEscape    *bool             `json:"aggressive_shell_escape"`
	TelegramChatTypeEnv      string            `json:"telegram_chat_type_env"`
	TelegramChatIDEnv        string            `json:"telegram_chat_id_env"`
	BackendTimeoutSeconds    *int              `json:"backend_timeout_seconds"`
}

func (c *Config) IsAggressiveShellEscape() bool {
	if c.AggressiveShellEscape == nil {
		return true // default true
	}
	return *c.AggressiveShellEscape
}

func (c *Config) GetBackendTimeout() int {
	if c.BackendTimeoutSeconds == nil {
		return 180 // default 3 minutes
	}
	return *c.BackendTimeoutSeconds
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("telegram_token is required")
	}
	if cfg.BackendCommand == "" {
		return nil, fmt.Errorf("backend_command is required")
	}

	return &cfg, nil
}

func (c *Config) IsUserAllowed(userID int64) bool {
	if !c.UserAllowlistRequired {
		return true
	}
	for _, id := range c.UserAllowlist {
		if id == userID {
			return true
		}
	}
	return false
}

func (c *Config) IsGroupAllowed(groupID int64) bool {
	if !c.GroupAllowlistRequired {
		return true
	}
	for _, id := range c.GroupAllowlist {
		if id == groupID {
			return true
		}
	}
	return false
}

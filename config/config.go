package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Config struct {
	TelegramToken            string                     `json:"telegram_token"`
	UserAllowlistRequired    bool                       `json:"user_allowlist_required"`
	GroupAllowlistRequired   bool                       `json:"group_allowlist_required"`
	UserAllowlist            []int64                    `json:"user_allowlist"`
	GroupAllowlist           []int64                    `json:"group_allowlist"`
	BackendCommand           string                     `json:"backend_command"`
	WorkingDirectory         string                     `json:"working_directory"`
	Environment              map[string]string          `json:"environment"`
	DropEnvironment          bool                       `json:"drop_environment"`
	EnableAttachments        bool                       `json:"enable_attachments"`
	AttachmentPath           string                     `json:"attachment_path"`
	AttachmentMethod         string                     `json:"attachment_method"`
	AttachmentPathChatPrefix string                     `json:"attachment_path_chat_prefix"`
	AttachmentPathChatIDSuffix bool                     `json:"attachment_path_chatid_suffix"`
	AggressiveShellEscape    *bool                      `json:"aggressive_shell_escape"`
	TelegramChatTypeEnv      string                     `json:"telegram_chat_type_env"`
	TelegramChatIDEnv        string                     `json:"telegram_chat_id_env"`
	BackendTimeoutSeconds    *int                       `json:"backend_timeout_seconds"`
	StripPrefix              *bool                      `json:"strip_prefix"`
	MessagePrefixOverrides   map[string]*ConfigOverride `json:"message_prefix_overrides"`
}

// ConfigOverride contains fields that can be overridden per-prefix.
// All fields are pointers to distinguish "not set" from "set to zero value".
type ConfigOverride struct {
	BackendCommand           *string           `json:"backend_command"`
	WorkingDirectory         *string           `json:"working_directory"`
	UserAllowlistRequired    *bool             `json:"user_allowlist_required"`
	UserAllowlist            []int64           `json:"user_allowlist"`
	GroupAllowlistRequired   *bool             `json:"group_allowlist_required"`
	GroupAllowlist           []int64           `json:"group_allowlist"`
	Environment              map[string]string `json:"environment"`
	DropEnvironment          *bool             `json:"drop_environment"`
	EnableAttachments        *bool             `json:"enable_attachments"`
	AttachmentPath           *string           `json:"attachment_path"`
	AttachmentMethod         *string           `json:"attachment_method"`
	AttachmentPathChatPrefix *string           `json:"attachment_path_chat_prefix"`
	AttachmentPathChatIDSuffix *bool           `json:"attachment_path_chatid_suffix"`
	AggressiveShellEscape    *bool             `json:"aggressive_shell_escape"`
	TelegramChatTypeEnv      *string           `json:"telegram_chat_type_env"`
	TelegramChatIDEnv        *string           `json:"telegram_chat_id_env"`
	BackendTimeoutSeconds    *int              `json:"backend_timeout_seconds"`
	StripPrefix              *bool             `json:"strip_prefix"`
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

func (c *Config) ShouldStripPrefix() bool {
	if c.StripPrefix == nil {
		return true // default true
	}
	return *c.StripPrefix
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

// IsUserAllowed checks if a user is allowed by this override's allowlist.
// Returns true if no user allowlist is required by the override.
func (o *ConfigOverride) IsUserAllowed(userID int64) bool {
	if o.UserAllowlistRequired == nil || !*o.UserAllowlistRequired {
		return true
	}
	for _, id := range o.UserAllowlist {
		if id == userID {
			return true
		}
	}
	return false
}

// IsGroupAllowed checks if a group is allowed by this override's allowlist.
// Returns true if no group allowlist is required by the override.
func (o *ConfigOverride) IsGroupAllowed(groupID int64) bool {
	if o.GroupAllowlistRequired == nil || !*o.GroupAllowlistRequired {
		return true
	}
	for _, id := range o.GroupAllowlist {
		if id == groupID {
			return true
		}
	}
	return false
}

// ApplyOverride creates a new Config with the override fields applied.
// The original config is not modified.
func (c *Config) ApplyOverride(override *ConfigOverride) *Config {
	if override == nil {
		return c
	}

	// Create a copy of the config
	newCfg := *c

	// Apply override fields if set
	if override.BackendCommand != nil {
		newCfg.BackendCommand = *override.BackendCommand
	}
	if override.WorkingDirectory != nil {
		newCfg.WorkingDirectory = *override.WorkingDirectory
	}
	if override.Environment != nil {
		newCfg.Environment = override.Environment
	}
	if override.DropEnvironment != nil {
		newCfg.DropEnvironment = *override.DropEnvironment
	}
	if override.EnableAttachments != nil {
		newCfg.EnableAttachments = *override.EnableAttachments
	}
	if override.AttachmentPath != nil {
		newCfg.AttachmentPath = *override.AttachmentPath
	}
	if override.AttachmentMethod != nil {
		newCfg.AttachmentMethod = *override.AttachmentMethod
	}
	if override.AttachmentPathChatPrefix != nil {
		newCfg.AttachmentPathChatPrefix = *override.AttachmentPathChatPrefix
	}
	if override.AttachmentPathChatIDSuffix != nil {
		newCfg.AttachmentPathChatIDSuffix = *override.AttachmentPathChatIDSuffix
	}
	if override.AggressiveShellEscape != nil {
		newCfg.AggressiveShellEscape = override.AggressiveShellEscape
	}
	if override.TelegramChatTypeEnv != nil {
		newCfg.TelegramChatTypeEnv = *override.TelegramChatTypeEnv
	}
	if override.TelegramChatIDEnv != nil {
		newCfg.TelegramChatIDEnv = *override.TelegramChatIDEnv
	}
	if override.BackendTimeoutSeconds != nil {
		newCfg.BackendTimeoutSeconds = override.BackendTimeoutSeconds
	}
	if override.StripPrefix != nil {
		newCfg.StripPrefix = override.StripPrefix
	}

	return &newCfg
}

// MatchResult contains the result of matching a message against prefix overrides.
type MatchResult struct {
	Prefix         string
	Override       *ConfigOverride
	StrippedText   string
	EffectiveConfig *Config
}

// MatchPrefixOverride finds the longest matching prefix override for a message.
// It checks the override's allowlists and returns the effective config.
// If no prefix matches or allowlists deny, returns the base config with original text.
func (c *Config) MatchPrefixOverride(msgText string, userID int64, chatID int64, chatType string) *MatchResult {
	result := &MatchResult{
		StrippedText:    msgText,
		EffectiveConfig: c,
	}

	if c.MessagePrefixOverrides == nil || len(c.MessagePrefixOverrides) == 0 {
		return result
	}

	// Get prefixes sorted by length (longest first)
	prefixes := make([]string, 0, len(c.MessagePrefixOverrides))
	for prefix := range c.MessagePrefixOverrides {
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})

	// Find the longest matching prefix
	for _, prefix := range prefixes {
		if !strings.HasPrefix(msgText, prefix) {
			continue
		}

		override := c.MessagePrefixOverrides[prefix]

		// Check override's allowlists
		if !override.IsUserAllowed(userID) {
			continue
		}

		// Check group allowlist only for group/supergroup chats
		if chatType == "group" || chatType == "supergroup" {
			if !override.IsGroupAllowed(chatID) {
				continue
			}
		}

		// Override is allowed, apply it
		result.Prefix = prefix
		result.Override = override
		result.EffectiveConfig = c.ApplyOverride(override)

		// Strip prefix if configured
		if result.EffectiveConfig.ShouldStripPrefix() {
			stripped := strings.TrimPrefix(msgText, prefix)
			// Also strip one leading space if present
			stripped = strings.TrimPrefix(stripped, " ")
			result.StrippedText = stripped
		}

		return result
	}

	return result
}

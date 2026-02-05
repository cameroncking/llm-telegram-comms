package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		errSubstr string
		validate  func(*Config) error
	}{
		{
			name: "valid minimal config",
			content: `{
				"telegram_token": "123:abc",
				"backend_command": "echo hello"
			}`,
			wantErr: false,
		},
		{
			name: "valid full config",
			content: `{
				"telegram_token": "123:abc",
				"backend_command": "./process.sh",
				"working_directory": "/tmp",
				"user_allowlist_required": true,
				"user_allowlist": [123, 456],
				"group_allowlist_required": true,
				"group_allowlist": [-100123, -100456],
				"environment": {"FOO": "bar"},
				"drop_environment": true
			}`,
			wantErr: false,
		},
		{
			name: "config with attachments enabled",
			content: `{
				"telegram_token": "123:abc",
				"backend_command": "echo",
				"enable_attachments": true,
				"attachment_path": "/tmp/attachments"
			}`,
			wantErr: false,
			validate: func(cfg *Config) error {
				if !cfg.EnableAttachments {
					return fmt.Errorf("expected EnableAttachments to be true")
				}
				if cfg.AttachmentPath != "/tmp/attachments" {
					return fmt.Errorf("expected AttachmentPath to be /tmp/attachments, got %s", cfg.AttachmentPath)
				}
				return nil
			},
		},
		{
			name: "attachments disabled by default",
			content: `{
				"telegram_token": "123:abc",
				"backend_command": "echo"
			}`,
			wantErr: false,
			validate: func(cfg *Config) error {
				if cfg.EnableAttachments {
					return fmt.Errorf("expected EnableAttachments to be false by default")
				}
				if cfg.AttachmentPath != "" {
					return fmt.Errorf("expected AttachmentPath to be empty by default")
				}
				return nil
			},
		},
		{
			name:      "missing token",
			content:   `{"backend_command": "echo"}`,
			wantErr:   true,
			errSubstr: "telegram_token is required",
		},
		{
			name:      "missing backend command",
			content:   `{"telegram_token": "123:abc"}`,
			wantErr:   true,
			errSubstr: "backend_command is required",
		},
		{
			name:      "invalid json",
			content:   `{invalid}`,
			wantErr:   true,
			errSubstr: "parsing config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := Load(configPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("config is nil")
			}
			if tt.validate != nil {
				if err := tt.validate(cfg); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestIsUserAllowed(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		userID   int64
		expected bool
	}{
		{
			name:     "allowlist not required",
			cfg:      Config{UserAllowlistRequired: false},
			userID:   123,
			expected: true,
		},
		{
			name:     "user in allowlist",
			cfg:      Config{UserAllowlistRequired: true, UserAllowlist: []int64{123, 456}},
			userID:   123,
			expected: true,
		},
		{
			name:     "user not in allowlist",
			cfg:      Config{UserAllowlistRequired: true, UserAllowlist: []int64{123, 456}},
			userID:   789,
			expected: false,
		},
		{
			name:     "empty allowlist with required",
			cfg:      Config{UserAllowlistRequired: true, UserAllowlist: []int64{}},
			userID:   123,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsUserAllowed(tt.userID); got != tt.expected {
				t.Errorf("IsUserAllowed(%d) = %v, want %v", tt.userID, got, tt.expected)
			}
		})
	}
}

func TestIsGroupAllowed(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		groupID  int64
		expected bool
	}{
		{
			name:     "allowlist not required",
			cfg:      Config{GroupAllowlistRequired: false},
			groupID:  -100123,
			expected: true,
		},
		{
			name:     "group in allowlist",
			cfg:      Config{GroupAllowlistRequired: true, GroupAllowlist: []int64{-100123, -100456}},
			groupID:  -100123,
			expected: true,
		},
		{
			name:     "group not in allowlist",
			cfg:      Config{GroupAllowlistRequired: true, GroupAllowlist: []int64{-100123, -100456}},
			groupID:  -100789,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IsGroupAllowed(tt.groupID); got != tt.expected {
				t.Errorf("IsGroupAllowed(%d) = %v, want %v", tt.groupID, got, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestConfigOverrideIsUserAllowed(t *testing.T) {
	tests := []struct {
		name     string
		override ConfigOverride
		userID   int64
		expected bool
	}{
		{
			name:     "nil UserAllowlistRequired",
			override: ConfigOverride{},
			userID:   123,
			expected: true,
		},
		{
			name:     "allowlist not required",
			override: ConfigOverride{UserAllowlistRequired: boolPtr(false)},
			userID:   123,
			expected: true,
		},
		{
			name:     "user in allowlist",
			override: ConfigOverride{UserAllowlistRequired: boolPtr(true), UserAllowlist: []int64{123, 456}},
			userID:   123,
			expected: true,
		},
		{
			name:     "user not in allowlist",
			override: ConfigOverride{UserAllowlistRequired: boolPtr(true), UserAllowlist: []int64{123, 456}},
			userID:   789,
			expected: false,
		},
		{
			name:     "empty allowlist with required",
			override: ConfigOverride{UserAllowlistRequired: boolPtr(true), UserAllowlist: []int64{}},
			userID:   123,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.override.IsUserAllowed(tt.userID); got != tt.expected {
				t.Errorf("ConfigOverride.IsUserAllowed(%d) = %v, want %v", tt.userID, got, tt.expected)
			}
		})
	}
}

func TestConfigOverrideIsGroupAllowed(t *testing.T) {
	tests := []struct {
		name     string
		override ConfigOverride
		groupID  int64
		expected bool
	}{
		{
			name:     "nil GroupAllowlistRequired",
			override: ConfigOverride{},
			groupID:  -100123,
			expected: true,
		},
		{
			name:     "allowlist not required",
			override: ConfigOverride{GroupAllowlistRequired: boolPtr(false)},
			groupID:  -100123,
			expected: true,
		},
		{
			name:     "group in allowlist",
			override: ConfigOverride{GroupAllowlistRequired: boolPtr(true), GroupAllowlist: []int64{-100123, -100456}},
			groupID:  -100123,
			expected: true,
		},
		{
			name:     "group not in allowlist",
			override: ConfigOverride{GroupAllowlistRequired: boolPtr(true), GroupAllowlist: []int64{-100123, -100456}},
			groupID:  -100789,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.override.IsGroupAllowed(tt.groupID); got != tt.expected {
				t.Errorf("ConfigOverride.IsGroupAllowed(%d) = %v, want %v", tt.groupID, got, tt.expected)
			}
		})
	}
}

func TestApplyOverride(t *testing.T) {
	baseCfg := &Config{
		TelegramToken:          "token",
		BackendCommand:         "default-backend",
		WorkingDirectory:       "/default",
		Environment:            map[string]string{"KEY": "value"},
		DropEnvironment:        false,
		EnableAttachments:      false,
		AttachmentPath:         "/attachments",
		AttachmentMethod:       "xml",
		BackendTimeoutSeconds:  intPtr(180),
	}

	tests := []struct {
		name     string
		override *ConfigOverride
		validate func(*Config) error
	}{
		{
			name:     "nil override returns same config",
			override: nil,
			validate: func(cfg *Config) error {
				if cfg.BackendCommand != "default-backend" {
					return fmt.Errorf("expected default-backend, got %s", cfg.BackendCommand)
				}
				return nil
			},
		},
		{
			name:     "override backend command",
			override: &ConfigOverride{BackendCommand: strPtr("new-backend")},
			validate: func(cfg *Config) error {
				if cfg.BackendCommand != "new-backend" {
					return fmt.Errorf("expected new-backend, got %s", cfg.BackendCommand)
				}
				if cfg.WorkingDirectory != "/default" {
					return fmt.Errorf("expected working directory to remain /default, got %s", cfg.WorkingDirectory)
				}
				return nil
			},
		},
		{
			name:     "override multiple fields",
			override: &ConfigOverride{
				BackendCommand:   strPtr("new-backend"),
				WorkingDirectory: strPtr("/new"),
				EnableAttachments: boolPtr(true),
				BackendTimeoutSeconds: intPtr(300),
			},
			validate: func(cfg *Config) error {
				if cfg.BackendCommand != "new-backend" {
					return fmt.Errorf("expected new-backend, got %s", cfg.BackendCommand)
				}
				if cfg.WorkingDirectory != "/new" {
					return fmt.Errorf("expected /new, got %s", cfg.WorkingDirectory)
				}
				if !cfg.EnableAttachments {
					return fmt.Errorf("expected EnableAttachments to be true")
				}
				if cfg.GetBackendTimeout() != 300 {
					return fmt.Errorf("expected timeout 300, got %d", cfg.GetBackendTimeout())
				}
				return nil
			},
		},
		{
			name:     "override environment replaces entirely",
			override: &ConfigOverride{
				Environment: map[string]string{"NEW_KEY": "new_value"},
			},
			validate: func(cfg *Config) error {
				if _, ok := cfg.Environment["KEY"]; ok {
					return fmt.Errorf("expected old KEY to be replaced")
				}
				if cfg.Environment["NEW_KEY"] != "new_value" {
					return fmt.Errorf("expected NEW_KEY=new_value")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := baseCfg.ApplyOverride(tt.override)
			if err := tt.validate(result); err != nil {
				t.Errorf("validation failed: %v", err)
			}
			// Ensure base config is not modified
			if baseCfg.BackendCommand != "default-backend" {
				t.Errorf("base config was modified")
			}
		})
	}
}

func TestMatchPrefixOverride(t *testing.T) {
	baseCfg := &Config{
		TelegramToken:  "token",
		BackendCommand: "default-backend",
		MessagePrefixOverrides: map[string]*ConfigOverride{
			"/research": {
				BackendCommand: strPtr("research-backend"),
			},
			"/code": {
				BackendCommand: strPtr("code-backend"),
			},
			"/re": {
				BackendCommand: strPtr("re-backend"),
			},
			"/premium": {
				BackendCommand:        strPtr("premium-backend"),
				UserAllowlistRequired: boolPtr(true),
				UserAllowlist:         []int64{111},
			},
			"/admin": {
				BackendCommand:         strPtr("admin-backend"),
				GroupAllowlistRequired: boolPtr(true),
				GroupAllowlist:         []int64{-100111},
			},
		},
	}

	tests := []struct {
		name            string
		msgText         string
		userID          int64
		chatID          int64
		chatType        string
		expectedPrefix  string
		expectedText    string
		expectedBackend string
	}{
		{
			name:            "no prefix match",
			msgText:         "hello world",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "",
			expectedText:    "hello world",
			expectedBackend: "default-backend",
		},
		{
			name:            "simple prefix match",
			msgText:         "/research What is AI?",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/research",
			expectedText:    "What is AI?",
			expectedBackend: "research-backend",
		},
		{
			name:            "longest prefix wins",
			msgText:         "/research topic",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/research",
			expectedText:    "topic",
			expectedBackend: "research-backend",
		},
		{
			name:            "shorter prefix when longer doesn't match",
			msgText:         "/re something",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/re",
			expectedText:    "something",
			expectedBackend: "re-backend",
		},
		{
			name:            "prefix only (no content after)",
			msgText:         "/code",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/code",
			expectedText:    "",
			expectedBackend: "code-backend",
		},
		{
			name:            "prefix with space only",
			msgText:         "/code ",
			userID:          123,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/code",
			expectedText:    "",
			expectedBackend: "code-backend",
		},
		{
			name:            "user allowed for premium",
			msgText:         "/premium request",
			userID:          111,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "/premium",
			expectedText:    "request",
			expectedBackend: "premium-backend",
		},
		{
			name:            "user not allowed for premium - fallback to default",
			msgText:         "/premium request",
			userID:          222,
			chatID:          456,
			chatType:        "private",
			expectedPrefix:  "",
			expectedText:    "/premium request",
			expectedBackend: "default-backend",
		},
		{
			name:            "group allowed for admin",
			msgText:         "/admin stats",
			userID:          123,
			chatID:          -100111,
			chatType:        "group",
			expectedPrefix:  "/admin",
			expectedText:    "stats",
			expectedBackend: "admin-backend",
		},
		{
			name:            "group not allowed for admin - fallback to default",
			msgText:         "/admin stats",
			userID:          123,
			chatID:          -100222,
			chatType:        "group",
			expectedPrefix:  "",
			expectedText:    "/admin stats",
			expectedBackend: "default-backend",
		},
		{
			name:            "admin in private chat - group allowlist skipped",
			msgText:         "/admin stats",
			userID:          123,
			chatID:          123,
			chatType:        "private",
			expectedPrefix:  "/admin",
			expectedText:    "stats",
			expectedBackend: "admin-backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := baseCfg.MatchPrefixOverride(tt.msgText, tt.userID, tt.chatID, tt.chatType)

			if result.Prefix != tt.expectedPrefix {
				t.Errorf("expected prefix %q, got %q", tt.expectedPrefix, result.Prefix)
			}
			if result.StrippedText != tt.expectedText {
				t.Errorf("expected stripped text %q, got %q", tt.expectedText, result.StrippedText)
			}
			if result.EffectiveConfig.BackendCommand != tt.expectedBackend {
				t.Errorf("expected backend %q, got %q", tt.expectedBackend, result.EffectiveConfig.BackendCommand)
			}
		})
	}
}

func TestMatchPrefixOverrideNoOverrides(t *testing.T) {
	cfg := &Config{
		TelegramToken:  "token",
		BackendCommand: "default-backend",
	}

	result := cfg.MatchPrefixOverride("/research hello", 123, 456, "private")

	if result.Prefix != "" {
		t.Errorf("expected no prefix match, got %q", result.Prefix)
	}
	if result.StrippedText != "/research hello" {
		t.Errorf("expected original text, got %q", result.StrippedText)
	}
	if result.EffectiveConfig.BackendCommand != "default-backend" {
		t.Errorf("expected default backend")
	}
}

func TestStripPrefixOption(t *testing.T) {
	tests := []struct {
		name           string
		globalStrip    *bool
		overrideStrip  *bool
		expectedText   string
	}{
		{
			name:          "default strips prefix",
			globalStrip:   nil,
			overrideStrip: nil,
			expectedText:  "hello",
		},
		{
			name:          "global strip false keeps prefix",
			globalStrip:   boolPtr(false),
			overrideStrip: nil,
			expectedText:  "/test hello",
		},
		{
			name:          "global strip true strips prefix",
			globalStrip:   boolPtr(true),
			overrideStrip: nil,
			expectedText:  "hello",
		},
		{
			name:          "override strip false overrides global true",
			globalStrip:   boolPtr(true),
			overrideStrip: boolPtr(false),
			expectedText:  "/test hello",
		},
		{
			name:          "override strip true overrides global false",
			globalStrip:   boolPtr(false),
			overrideStrip: boolPtr(true),
			expectedText:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TelegramToken:  "token",
				BackendCommand: "default-backend",
				StripPrefix:    tt.globalStrip,
				MessagePrefixOverrides: map[string]*ConfigOverride{
					"/test": {
						BackendCommand: strPtr("test-backend"),
						StripPrefix:    tt.overrideStrip,
					},
				},
			}

			result := cfg.MatchPrefixOverride("/test hello", 123, 456, "private")

			if result.StrippedText != tt.expectedText {
				t.Errorf("expected text %q, got %q", tt.expectedText, result.StrippedText)
			}
		})
	}
}

func TestShouldStripPrefix(t *testing.T) {
	tests := []struct {
		name     string
		strip    *bool
		expected bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{StripPrefix: tt.strip}
			if got := cfg.ShouldStripPrefix(); got != tt.expected {
				t.Errorf("ShouldStripPrefix() = %v, want %v", got, tt.expected)
			}
		})
	}
}

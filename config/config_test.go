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

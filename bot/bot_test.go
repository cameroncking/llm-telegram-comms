package bot

import (
	"testing"

	"github.com/go-telegram/bot/models"

	"github.com/exedev/llm-telegram-comms/config"
)

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		msg      *models.Message
		expected bool
	}{
		{
			name: "no restrictions - private chat",
			cfg: &config.Config{
				UserAllowlistRequired:  false,
				GroupAllowlistRequired: false,
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 123, Type: "private"},
			},
			expected: true,
		},
		{
			name: "no restrictions - group chat",
			cfg: &config.Config{
				UserAllowlistRequired:  false,
				GroupAllowlistRequired: false,
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "group"},
			},
			expected: true,
		},
		{
			name: "user allowlist - user allowed",
			cfg: &config.Config{
				UserAllowlistRequired: true,
				UserAllowlist:         []int64{123, 456},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 123, Type: "private"},
			},
			expected: true,
		},
		{
			name: "user allowlist - user not allowed",
			cfg: &config.Config{
				UserAllowlistRequired: true,
				UserAllowlist:         []int64{456, 789},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 123, Type: "private"},
			},
			expected: false,
		},
		{
			name: "group allowlist - group allowed",
			cfg: &config.Config{
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{-100456},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "group"},
			},
			expected: true,
		},
		{
			name: "group allowlist - group not allowed",
			cfg: &config.Config{
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{-100789},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "group"},
			},
			expected: false,
		},
		{
			name: "supergroup allowlist check",
			cfg: &config.Config{
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{-100456},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "supergroup"},
			},
			expected: true,
		},
		{
			name: "group allowlist not applied to private chat",
			cfg: &config.Config{
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 123, Type: "private"},
			},
			expected: true,
		},
		{
			name: "both allowlists - both allowed",
			cfg: &config.Config{
				UserAllowlistRequired:  true,
				UserAllowlist:          []int64{123},
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{-100456},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "group"},
			},
			expected: true,
		},
		{
			name: "both allowlists - user not allowed",
			cfg: &config.Config{
				UserAllowlistRequired:  true,
				UserAllowlist:          []int64{456},
				GroupAllowlistRequired: true,
				GroupAllowlist:         []int64{-100456},
			},
			msg: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: -100456, Type: "group"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bot{cfg: tt.cfg}
			if got := b.isAllowed(tt.msg); got != tt.expected {
				t.Errorf("isAllowed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncate(tt.input, tt.maxLen); got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestIsAllowedWithAttachmentConfig(t *testing.T) {
	// Test that attachment config doesn't affect allowlist behavior
	cfg := &config.Config{
		UserAllowlistRequired:  true,
		UserAllowlist:          []int64{123},
		GroupAllowlistRequired: false,
		EnableAttachments:      true,
		AttachmentPath:         "/tmp/attachments",
	}

	b := &Bot{cfg: cfg}

	msg := &models.Message{
		From: &models.User{ID: 123},
		Chat: models.Chat{ID: 123, Type: "private"},
	}

	if !b.isAllowed(msg) {
		t.Error("expected user 123 to be allowed")
	}

	msg.From.ID = 456
	if b.isAllowed(msg) {
		t.Error("expected user 456 to be rejected")
	}
}

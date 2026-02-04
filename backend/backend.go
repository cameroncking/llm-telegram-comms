package backend

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/exedev/llm-telegram-comms/config"
)

type Backend struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Backend {
	return &Backend{cfg: cfg}
}

type ExecOptions struct {
	ChatType string // "user" or "group"
	ChatID   int64
}

func (b *Backend) Execute(ctx context.Context, input string, opts *ExecOptions, extraArgs ...string) (string, error) {
	// Apply timeout to the context
	timeoutSecs := b.cfg.GetBackendTimeout()
	if timeoutSecs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
		defer cancel()
	}

	command := b.cfg.BackendCommand
	if len(extraArgs) > 0 {
		aggressive := b.cfg.IsAggressiveShellEscape()
		for _, arg := range extraArgs {
			if aggressive {
				command += " " + aggressiveShellEscape(arg)
			} else {
				command += " " + shellEscape(arg)
			}
		}
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	if b.cfg.WorkingDirectory != "" {
		cmd.Dir = b.cfg.WorkingDirectory
	}

	cmd.Env = b.buildEnv(opts)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %d seconds", b.cfg.GetBackendTimeout())
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
		}
		return "", fmt.Errorf("command failed: %w", err)
	}

	return stdout.String(), nil
}

func shellEscape(s string) string {
	// Simple shell escaping by wrapping in single quotes
	// and escaping any single quotes within
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func aggressiveShellEscape(s string) string {
	// Aggressive shell escaping: only allow known safe printable characters
	// Safe characters: a-z A-Z 0-9 . _ - / 
	// Unsafe characters are replaced with _
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == '-' || r == '/' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	return "'" + result.String() + "'"
}

func (b *Backend) buildEnv(opts *ExecOptions) []string {
	var env []string

	if !b.cfg.DropEnvironment {
		env = os.Environ()
	}

	for k, v := range b.cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts != nil {
		if b.cfg.TelegramChatTypeEnv != "" && opts.ChatType != "" {
			env = append(env, fmt.Sprintf("%s=%s", b.cfg.TelegramChatTypeEnv, opts.ChatType))
		}
		if b.cfg.TelegramChatIDEnv != "" && opts.ChatID != 0 {
			env = append(env, fmt.Sprintf("%s=%d", b.cfg.TelegramChatIDEnv, opts.ChatID))
		}
	}

	return env
}

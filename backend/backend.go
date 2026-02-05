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

type ExecOptions struct {
	ChatType string // "user" or "group"
	ChatID   int64
}

// Execute runs the backend command with the given input and config.
func Execute(ctx context.Context, input string, cfg *config.Config, opts *ExecOptions, extraArgs ...string) (string, error) {
	// Apply timeout to the context
	timeoutSecs := cfg.GetBackendTimeout()
	if timeoutSecs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
		defer cancel()
	}

	command := cfg.BackendCommand
	if len(extraArgs) > 0 {
		aggressive := cfg.IsAggressiveShellEscape()
		for _, arg := range extraArgs {
			if aggressive {
				command += " " + aggressiveShellEscape(arg)
			} else {
				command += " " + shellEscape(arg)
			}
		}
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	if cfg.WorkingDirectory != "" {
		cmd.Dir = cfg.WorkingDirectory
	}

	cmd.Env = buildEnv(cfg, opts)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %d seconds", cfg.GetBackendTimeout())
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

func buildEnv(cfg *config.Config, opts *ExecOptions) []string {
	var env []string

	if !cfg.DropEnvironment {
		env = os.Environ()
	}

	for k, v := range cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts != nil {
		if cfg.TelegramChatTypeEnv != "" && opts.ChatType != "" {
			env = append(env, fmt.Sprintf("%s=%s", cfg.TelegramChatTypeEnv, opts.ChatType))
		}
		if cfg.TelegramChatIDEnv != "" && opts.ChatID != 0 {
			env = append(env, fmt.Sprintf("%s=%d", cfg.TelegramChatIDEnv, opts.ChatID))
		}
	}

	return env
}

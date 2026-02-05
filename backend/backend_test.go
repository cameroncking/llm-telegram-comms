package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/exedev/llm-telegram-comms/config"
)

// Re-export for testing
var shellEscapeTest = shellEscape
var aggressiveShellEscapeTest = aggressiveShellEscape

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		input      string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "echo command",
			cfg: &config.Config{
				BackendCommand: "cat",
			},
			input:      "hello world",
			wantOutput: "hello world",
			wantErr:    false,
		},
		{
			name: "transform input",
			cfg: &config.Config{
				BackendCommand: "tr a-z A-Z",
			},
			input:      "hello",
			wantOutput: "HELLO",
			wantErr:    false,
		},
		{
			name: "failing command",
			cfg: &config.Config{
				BackendCommand: "exit 1",
			},
			input:   "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			output, err := Execute(ctx, tt.input, tt.cfg, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if output != tt.wantOutput {
				t.Errorf("output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestExecuteWithWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		BackendCommand:   "ls",
		WorkingDirectory: tmpDir,
	}

	output, err := Execute(context.Background(), "", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "test.txt") {
		t.Errorf("expected output to contain test.txt, got %q", output)
	}
}

func TestExecuteWithEnvironment(t *testing.T) {
	cfg := &config.Config{
		BackendCommand: "echo $MY_VAR",
		Environment: map[string]string{
			"MY_VAR": "custom_value",
		},
	}

	output, err := Execute(context.Background(), "", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(output) != "custom_value" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(output), "custom_value")
	}
}

func TestExecuteDropEnvironment(t *testing.T) {
	// Set an environment variable that should be dropped
	os.Setenv("TEST_INHERITED_VAR", "should_be_dropped")
	defer os.Unsetenv("TEST_INHERITED_VAR")

	cfg := &config.Config{
		BackendCommand:  "echo \"inherited:$TEST_INHERITED_VAR custom:$CUSTOM_VAR\"",
		DropEnvironment: true,
		Environment: map[string]string{
			"CUSTOM_VAR": "present",
		},
	}

	output, err := Execute(context.Background(), "", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The inherited var should be empty, custom should be present
	if !strings.Contains(output, "custom:present") {
		t.Errorf("expected custom var to be present, got %q", output)
	}
	if strings.Contains(output, "should_be_dropped") {
		t.Errorf("inherited var should have been dropped, got %q", output)
	}
}

func TestExecuteContextCancellation(t *testing.T) {
	cfg := &config.Config{
		BackendCommand: "sleep 10",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := Execute(ctx, "", cfg, nil)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func TestExecuteWithExtraArgs(t *testing.T) {
	cfg := &config.Config{
		BackendCommand: "echo",
	}

	output, err := Execute(context.Background(), "", cfg, nil, "-a", "file1.png", "-a", "file2.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The echo command should output the arguments
	if !strings.Contains(output, "-a") {
		t.Errorf("expected output to contain '-a', got %q", output)
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "file.png",
			expected: "'file.png'",
		},
		{
			name:     "filename with spaces",
			input:    "my file.png",
			expected: "'my file.png'",
		},
		{
			name:     "filename with single quote",
			input:    "it's a file.png",
			expected: "'it'\"'\"'s a file.png'",
		},
		{
			name:     "filename with double quotes",
			input:    `file "name".png`,
			expected: `'file "name".png'`,
		},
		{
			name:     "filename with backticks",
			input:    "file`name`.png",
			expected: "'file`name`.png'",
		},
		{
			name:     "filename with dollar sign",
			input:    "file$name.png",
			expected: "'file$name.png'",
		},
		{
			name:     "filename with newline",
			input:    "file\nname.png",
			expected: "'file\nname.png'",
		},
		{
			name:     "filename with semicolon",
			input:    "file;rm -rf /.png",
			expected: "'file;rm -rf /.png'",
		},
		{
			name:     "filename with pipe",
			input:    "file|cat /etc/passwd.png",
			expected: "'file|cat /etc/passwd.png'",
		},
		{
			name:     "path with directory",
			input:    "/data/attachments/file.png",
			expected: "'/data/attachments/file.png'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellEscape(tt.input)
			if got != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShellEscapeExecution(t *testing.T) {
	// Test that shell-escaped strings actually work correctly when executed
	// Uses non-aggressive mode to preserve special characters
	tests := []struct {
		name     string
		filename string
	}{
		{"simple", "file.png"},
		{"with spaces", "my file.png"},
		{"with single quote", "it's a file.png"},
		{"with double quotes", `file "name".png`},
		{"with special chars", "file$`name;|&.png"},
		{"with parentheses", "file(1).png"},
		{"with brackets", "file[1].png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggressive := false
			cfg := &config.Config{
				BackendCommand:        "printf '%s'",
				AggressiveShellEscape: &aggressive,
			}

			output, err := Execute(context.Background(), "", cfg, nil, tt.filename)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The output should contain the filename as-is
			if !strings.Contains(output, tt.filename) {
				t.Errorf("expected output to contain %q, got %q", tt.filename, output)
			}
		})
	}
}

func TestShellEscapePreventInjection(t *testing.T) {
	// Test that malicious filenames don't execute arbitrary commands
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "pwned")

	maliciousNames := []string{
		"'; touch " + markerFile + " #",
		"\"; touch " + markerFile + " #",
		"$(touch " + markerFile + ")",
		"`touch " + markerFile + "`",
		"| touch " + markerFile,
		"; touch " + markerFile,
		"&& touch " + markerFile,
	}

	for _, name := range maliciousNames {
		t.Run(name, func(t *testing.T) {
			// Remove marker file if it exists
			os.Remove(markerFile)

			aggressive := false
			cfg := &config.Config{
				BackendCommand:        "echo test",
				AggressiveShellEscape: &aggressive,
			}

			_, err := Execute(context.Background(), "", cfg, nil, "-a", name)
			if err != nil {
				// Error is acceptable, injection is not
			}

			// Check that marker file was NOT created
			if _, err := os.Stat(markerFile); err == nil {
				t.Errorf("shell injection succeeded with filename %q", name)
			}
		})
	}
}

func TestAggressiveShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "file.png",
			expected: "'file.png'",
		},
		{
			name:     "filename with spaces replaced",
			input:    "my file.png",
			expected: "'my_file.png'",
		},
		{
			name:     "filename with special chars replaced",
			input:    "file$`';|&.png",
			expected: "'file______.png'",
		},
		{
			name:     "path preserved",
			input:    "/data/attachments/file.png",
			expected: "'/data/attachments/file.png'",
		},
		{
			name:     "underscore and dash preserved",
			input:    "my_file-name.png",
			expected: "'my_file-name.png'",
		},
		{
			name:     "injection attempt sanitized",
			input:    "'; rm -rf /",
			expected: "'___rm_-rf_/'",
		},
		{
			name:     "command substitution sanitized",
			input:    "$(whoami).png",
			expected: "'__whoami_.png'",
		},
		{
			name:     "backtick substitution sanitized",
			input:    "`whoami`.png",
			expected: "'_whoami_.png'",
		},
		{
			name:     "timestamp filename preserved",
			input:    "2026-02-04T14-30-25Z_IMG_9812.png",
			expected: "'2026-02-04T14-30-25Z_IMG_9812.png'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggressiveShellEscape(tt.input)
			if got != tt.expected {
				t.Errorf("aggressiveShellEscape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAggressiveShellEscapePreventInjection(t *testing.T) {
	// Test that malicious filenames don't execute arbitrary commands with aggressive mode
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "pwned")

	maliciousNames := []string{
		"'; touch " + markerFile + " #",
		"\"; touch " + markerFile + " #",
		"$(touch " + markerFile + ")",
		"`touch " + markerFile + "`",
		"| touch " + markerFile,
		"; touch " + markerFile,
		"&& touch " + markerFile,
	}

	for _, name := range maliciousNames {
		t.Run(name, func(t *testing.T) {
			// Remove marker file if it exists
			os.Remove(markerFile)

			// aggressive_shell_escape defaults to true
			cfg := &config.Config{
				BackendCommand: "echo test",
			}

			_, err := Execute(context.Background(), "", cfg, nil, "-a", name)
			if err != nil {
				// Error is acceptable, injection is not
			}

			// Check that marker file was NOT created
			if _, err := os.Stat(markerFile); err == nil {
				t.Errorf("shell injection succeeded with filename %q", name)
			}
		})
	}
}

func TestAggressiveShellEscapeDefault(t *testing.T) {
	// Test that aggressive shell escape is enabled by default
	cfg := &config.Config{
		BackendCommand: "echo",
	}

	if !cfg.IsAggressiveShellEscape() {
		t.Error("expected IsAggressiveShellEscape() to return true by default")
	}

	// Test explicit false
	aggressiveFalse := false
	cfg.AggressiveShellEscape = &aggressiveFalse
	if cfg.IsAggressiveShellEscape() {
		t.Error("expected IsAggressiveShellEscape() to return false when set to false")
	}

	// Test explicit true
	aggressiveTrue := true
	cfg.AggressiveShellEscape = &aggressiveTrue
	if !cfg.IsAggressiveShellEscape() {
		t.Error("expected IsAggressiveShellEscape() to return true when set to true")
	}
}

func TestExecuteWithChatTypeEnv(t *testing.T) {
	cfg := &config.Config{
		BackendCommand:      "echo $CHAT_TYPE",
		TelegramChatTypeEnv: "CHAT_TYPE",
	}

	// Test user chat type
	output, err := Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "user", ChatID: 123})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "user" {
		t.Errorf("expected 'user', got %q", strings.TrimSpace(output))
	}

	// Test group chat type
	output, err = Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "group", ChatID: -100123})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "group" {
		t.Errorf("expected 'group', got %q", strings.TrimSpace(output))
	}
}

func TestExecuteWithChatIDEnv(t *testing.T) {
	cfg := &config.Config{
		BackendCommand:    "echo $CHAT_ID",
		TelegramChatIDEnv: "CHAT_ID",
	}

	// Test positive ID (user)
	output, err := Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "user", ChatID: 123456789})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "123456789" {
		t.Errorf("expected '123456789', got %q", strings.TrimSpace(output))
	}

	// Test negative ID (group)
	output, err = Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "group", ChatID: -100123456789})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "-100123456789" {
		t.Errorf("expected '-100123456789', got %q", strings.TrimSpace(output))
	}
}

func TestExecuteWithBothChatEnvVars(t *testing.T) {
	cfg := &config.Config{
		BackendCommand:      "echo \"type:$CHAT_TYPE id:$CHAT_ID\"",
		TelegramChatTypeEnv: "CHAT_TYPE",
		TelegramChatIDEnv:   "CHAT_ID",
	}

	output, err := Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "user", ChatID: 12345})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "type:user id:12345" {
		t.Errorf("expected 'type:user id:12345', got %q", strings.TrimSpace(output))
	}
}

func TestExecuteWithChatEnvVarsNotSet(t *testing.T) {
	// When env var names are not configured, the vars should not be set
	cfg := &config.Config{
		BackendCommand: "echo \"type:$CHAT_TYPE id:$CHAT_ID\"",
		// TelegramChatTypeEnv and TelegramChatIDEnv not set
	}

	output, err := Execute(context.Background(), "", cfg, &ExecOptions{ChatType: "user", ChatID: 12345})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Variables should be empty since config doesn't define them
	if strings.TrimSpace(output) != "type: id:" {
		t.Errorf("expected 'type: id:', got %q", strings.TrimSpace(output))
	}
}

func TestExecuteWithNilExecOptions(t *testing.T) {
	cfg := &config.Config{
		BackendCommand:      "echo \"type:$CHAT_TYPE id:$CHAT_ID\"",
		TelegramChatTypeEnv: "CHAT_TYPE",
		TelegramChatIDEnv:   "CHAT_ID",
	}

	// With nil opts, env vars should not be set
	output, err := Execute(context.Background(), "", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "type: id:" {
		t.Errorf("expected 'type: id:', got %q", strings.TrimSpace(output))
	}
}

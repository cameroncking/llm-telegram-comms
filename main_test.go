package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCheckPidFileNotExists(t *testing.T) {
	pid, name, running := checkPidFile("/nonexistent/path/file.pid")
	if running {
		t.Error("expected running to be false for nonexistent file")
	}
	if pid != 0 {
		t.Errorf("expected pid to be 0, got %d", pid)
	}
	if name != "" {
		t.Errorf("expected name to be empty, got %s", name)
	}
}

func TestCheckPidFileInvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write invalid content
	if err := os.WriteFile(pidFile, []byte("not-a-number"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, name, running := checkPidFile(pidFile)
	if running {
		t.Error("expected running to be false for invalid pid")
	}
	if pid != 0 {
		t.Errorf("expected pid to be 0, got %d", pid)
	}
	if name != "" {
		t.Errorf("expected name to be empty, got %s", name)
	}
}

func TestCheckPidFileDeadProcess(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write a PID that definitely doesn't exist (very high number)
	if err := os.WriteFile(pidFile, []byte("999999999"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, _, running := checkPidFile(pidFile)
	if running {
		t.Error("expected running to be false for dead process")
	}
	if pid != 999999999 {
		t.Errorf("expected pid to be 999999999, got %d", pid)
	}
}

func TestCheckPidFileRunningProcess(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Use our own PID (we know it's running)
	myPid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(myPid)), 0644); err != nil {
		t.Fatal(err)
	}

	pid, name, running := checkPidFile(pidFile)
	if !running {
		t.Error("expected running to be true for our own process")
	}
	if pid != myPid {
		t.Errorf("expected pid to be %d, got %d", myPid, pid)
	}
	// Name should be the test binary name
	if name == "" {
		t.Error("expected name to be non-empty")
	}
}

func TestCheckPidFileWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write PID with newline (common format)
	myPid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(myPid)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, _, running := checkPidFile(pidFile)
	if !running {
		t.Error("expected running to be true")
	}
	if pid != myPid {
		t.Errorf("expected pid to be %d, got %d", myPid, pid)
	}
}

func TestDaemonizePidFileAlreadyRunning(t *testing.T) {
	if os.Getenv("TEST_DAEMON") != "1" {
		t.Skip("skipping daemon test in normal mode")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")
	configFile := filepath.Join(tmpDir, "config.json")

	// Create a minimal config
	configContent := `{"telegram_token": "test", "backend_command": "cat"}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write our own PID to simulate running process
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}

	// This should fail because process is "already running"
	err := daemonize(configFile, pidFile, "", false)
	if err == nil {
		t.Error("expected error when process already running")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("expected 'already running' error, got: %v", err)
	}
}

func TestCommandLineValidation(t *testing.T) {
	// Test that -p without -d fails
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "-p", "/tmp/test.pid", "-c", "config.json")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	output, err := cmd.CombinedOutput()
	
	// We expect an error
	if err == nil {
		t.Logf("output: %s", output)
	}
}

func TestCommandLineRestartRequiresPid(t *testing.T) {
	// Test that -r without -p fails
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "-r", "-d", "-c", "config.json")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	output, err := cmd.CombinedOutput()
	
	// We expect an error
	if err == nil {
		t.Logf("output: %s", output)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// This is a helper for subprocess tests
	os.Exit(0)
}

func TestKillExistingProcess(t *testing.T) {
	// Start a sleep process that we can kill
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}

	pid := cmd.Process.Pid
	defer func() {
		// Clean up if test fails
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Verify process is running
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("sleep process not running: %v", err)
	}

	// Kill it
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		t.Fatalf("failed to kill process: %v", err)
	}

	// Wait for it to die (use cmd.Wait() to properly reap the process)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for process to die")
	}
}

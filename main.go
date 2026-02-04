package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/exedev/llm-telegram-comms/bot"
	"github.com/exedev/llm-telegram-comms/config"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.StringVar(configPath, "c", "", "Path to config file (shorthand)")

	daemon := flag.Bool("daemon", false, "Fork the process in the background")
	flag.BoolVar(daemon, "d", false, "Fork the process in the background (shorthand)")

	pidFile := flag.String("pid", "", "Write PID to file (requires -d)")
	flag.StringVar(pidFile, "p", "", "Write PID to file (shorthand, requires -d)")

	logFile := flag.String("log", "", "Write logs to file instead of stdout")
	flag.StringVar(logFile, "l", "", "Write logs to file (shorthand)")

	restart := flag.Bool("restart", false, "Kill existing process from PID file before starting (requires -p)")
	flag.BoolVar(restart, "r", false, "Kill existing process from PID file before starting (shorthand, requires -p)")

	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: config file path is required")
		fmt.Fprintln(os.Stderr, "Usage: llm-telegram-comms -c <config.json>")
		os.Exit(1)
	}

	if *pidFile != "" && !*daemon {
		fmt.Fprintln(os.Stderr, "Error: -p/--pid requires -d/--daemon")
		os.Exit(1)
	}

	if *restart && *pidFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -r/--restart requires -p/--pid")
		os.Exit(1)
	}

	if *daemon {
		if err := daemonize(*configPath, *pidFile, *logFile, *restart); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	run(*configPath, *logFile)
}

func daemonize(configPath, pidFile, logFile string, restart bool) error {
	myName := filepath.Base(os.Args[0])

	if pidFile != "" {
		existingPid, procName, running := checkPidFile(pidFile)
		if running {
			if procName == myName {
				if restart {
					// Kill the existing process
					if err := syscall.Kill(existingPid, syscall.SIGTERM); err != nil {
						return fmt.Errorf("failed to kill existing process %d: %w", existingPid, err)
					}
					fmt.Printf("Killed existing process %d\n", existingPid)
				} else {
					return fmt.Errorf("process already running with PID %d", existingPid)
				}
			}
			// If different name, we proceed and overwrite the PID file
		}
	}

	args := []string{os.Args[0], "-c", configPath}
	if logFile != "" {
		args = append(args, "-l", logFile)
	}

	proc, err := os.StartProcess(os.Args[0], args, &os.ProcAttr{
		Dir: "",
		Env: os.Environ(),
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("starting daemon process: %w", err)
	}

	if pidFile != "" {
		if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", proc.Pid)), 0644); err != nil {
			return fmt.Errorf("writing pid file: %w", err)
		}
	}

	fmt.Printf("Started daemon with PID %d\n", proc.Pid)
	return nil
}

// checkPidFile returns (pid, process name, is running)
func checkPidFile(pidFile string) (int, string, bool) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, "", false
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, "", false
	}

	// Check if process is running
	if err := syscall.Kill(pid, 0); err != nil {
		return pid, "", false
	}

	// Get process name from /proc/exe (symlink to actual executable)
	exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		// Process exists but can't read name, assume it's running
		return pid, "", true
	}

	return pid, filepath.Base(exePath), true
}

func run(configPath, logFile string) {
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	b.Start(ctx)
}

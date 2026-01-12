package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gojekfarm/xtools/errors"
)

const (
	daemonChildEnvVar     = "XDB_DAEMON_CHILD"
	healthCheckTimeout    = 3 * time.Second
	stopTimeout           = 30 * time.Second
	stopPollInterval      = 100 * time.Millisecond
	healthCheckPollDelay  = 100 * time.Millisecond
	logFollowPollInterval = 500 * time.Millisecond
)

// Daemon manages the XDB daemon lifecycle and operations.
type Daemon struct {
	Config *Config
}

// NewDaemon creates a new Daemon instance with the given configuration.
func NewDaemon(cfg *Config) *Daemon {
	return &Daemon{
		Config: cfg,
	}
}

// DaemonStatusInfo holds information about the daemon status.
type DaemonStatusInfo struct {
	Status         string   `json:"status"`
	PID            int      `json:"pid,omitempty"`
	Address        string   `json:"address,omitempty"`
	Addresses      []string `json:"addresses,omitempty"`
	Healthy        bool     `json:"healthy"`
	ResponseTimeMs int64    `json:"response_time_ms,omitempty"`
}

func (d *Daemon) readPID() (int, error) {
	pidPath := d.Config.PIDFile()
	data, err := os.ReadFile(pidPath) // #nosec G304 - pidPath is from trusted config
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.Wrap(err, "path", pidPath)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, errors.Wrap(err, "path", pidPath, "value", string(data))
	}

	return pid, nil
}

func (d *Daemon) writePID(pid int) error {
	pidPath := d.Config.PIDFile()
	pidDir := filepath.Dir(pidPath)

	if err := os.MkdirAll(pidDir, 0o700); err != nil {
		return errors.Wrap(err, "path", pidDir)
	}

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600); err != nil {
		return errors.Wrap(err, "path", pidPath)
	}

	return nil
}

func (d *Daemon) removePID() error {
	pidPath := d.Config.PIDFile()
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "path", pidPath)
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID exists.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Start starts the daemon process.
func (d *Daemon) Start() error {
	if os.Getenv(daemonChildEnvVar) == "1" {
		return d.runChild()
	}
	return d.spawnParent()
}

func (d *Daemon) spawnParent() error {
	pid, err := d.readPID()
	if err != nil {
		return err
	}

	if pid > 0 && IsProcessRunning(pid) {
		return errors.Wrap(ErrDaemonAlreadyRunning, "pid", strconv.Itoa(pid))
	}

	if pid > 0 {
		fmt.Printf("Warning: Found stale PID file (process %d not running)\n", pid)
		fmt.Println("Cleaning up and starting daemon...")
		if err := d.removePID(); err != nil {
			return errors.Wrap(err, "path", d.Config.PIDFile())
		}
	} else {
		fmt.Println("Starting XDB daemon...")
	}

	logDir := filepath.Dir(d.Config.LogFile())
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return errors.Wrap(err, "path", logDir)
	}

	logFile, err := os.OpenFile(d.Config.LogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) // #nosec G304 - path from trusted config
	if err != nil {
		return errors.Wrap(err, "path", d.Config.LogFile())
	}

	executable, err := os.Executable()
	if err != nil {
		_ = logFile.Close()
		return err
	}

	cmd := exec.CommandContext(context.Background(), executable, "daemon", "start") // #nosec G204 - executable is from os.Executable()
	cmd.Env = append(os.Environ(), daemonChildEnvVar+"=1")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = "/"

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}

	_ = logFile.Close()

	if err := d.waitForHealthy(context.Background(), healthCheckTimeout); err != nil {
		newPID, _ := d.readPID()
		return errors.Wrap(err, "pid", strconv.Itoa(newPID))
	}

	newPID, _ := d.readPID()
	fmt.Printf("Daemon started successfully (PID: %d)\n", newPID)
	d.printListeningAddresses()
	return nil
}

func (d *Daemon) printListeningAddresses() {
	if d.Config.Daemon.Addr != "" && d.Config.Daemon.Socket != "" {
		fmt.Printf("Server listening on %s and %s\n", d.Config.Daemon.Addr, d.Config.SocketPath())
	} else if d.Config.Daemon.Addr != "" {
		fmt.Printf("Server listening on %s\n", d.Config.Daemon.Addr)
	} else {
		fmt.Printf("Server listening on %s\n", d.Config.SocketPath())
	}
}

func (d *Daemon) runChild() error {
	if err := d.writePID(os.Getpid()); err != nil {
		return errors.Wrap(err, "path", d.Config.PIDFile())
	}

	server, err := NewServer(d.Config)
	if err != nil {
		_ = d.removePID()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := server.Run(ctx); err != nil {
		_ = d.removePID()
		return err
	}

	_ = d.removePID()
	return nil
}

func (d *Daemon) waitForHealthy(ctx context.Context, timeout time.Duration) error {
	if d.Config.Daemon.Addr == "" {
		return nil
	}

	deadline := time.Now().Add(timeout)
	healthURL := fmt.Sprintf("http://%s/v1/schemas", d.Config.Daemon.Addr)

	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return errors.Wrap(err, "url", healthURL)
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(healthCheckPollDelay)
	}

	return errors.Wrap(ErrHealthCheckTimeout, "timeout", timeout.String())
}

// Stop stops the running daemon.
func (d *Daemon) Stop(force bool) error {
	pid, err := d.readPID()
	if err != nil {
		return err
	}

	if pid == 0 {
		fmt.Println("Daemon is not running")
		return nil
	}

	if !IsProcessRunning(pid) {
		if err := d.removePID(); err != nil {
			return errors.Wrap(err, "path", d.Config.PIDFile())
		}
		fmt.Println("Daemon is not running (cleaned up stale PID file)")
		return nil
	}

	fmt.Printf("Stopping XDB daemon (PID: %d)...\n", pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrap(err, "pid", strconv.Itoa(pid))
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return errors.Wrap(err, "pid", strconv.Itoa(pid))
	}

	stopped := waitForProcessStop(pid, stopTimeout)

	if !stopped {
		if force {
			if err := process.Signal(syscall.SIGKILL); err != nil {
				return errors.Wrap(err, "pid", strconv.Itoa(pid))
			}
			waitForProcessStop(pid, 5*time.Second)
		} else {
			return errors.Wrap(ErrStopTimeout, "timeout", stopTimeout.String(), "pid", strconv.Itoa(pid))
		}
	}

	if err := d.removePID(); err != nil {
		return errors.Wrap(err, "path", d.Config.PIDFile())
	}

	fmt.Println("Daemon stopped successfully")
	return nil
}

func waitForProcessStop(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if !IsProcessRunning(pid) {
			return true
		}
		time.Sleep(stopPollInterval)
	}

	return !IsProcessRunning(pid)
}

// GetStatus returns the current daemon status.
func (d *Daemon) GetStatus(ctx context.Context) (*DaemonStatusInfo, error) {
	info := &DaemonStatusInfo{
		Address: d.Config.Daemon.Addr,
	}

	info.Addresses = []string{}
	if d.Config.Daemon.Addr != "" {
		info.Addresses = append(info.Addresses, fmt.Sprintf("tcp://%s", d.Config.Daemon.Addr))
	}
	if d.Config.Daemon.Socket != "" {
		info.Addresses = append(info.Addresses, fmt.Sprintf("unix://%s", d.Config.SocketPath()))
	}

	pid, err := d.readPID()
	if err != nil {
		return nil, err
	}

	if pid == 0 || !IsProcessRunning(pid) {
		info.Status = "stopped"
		return info, nil
	}

	info.PID = pid
	info.Status = "running"

	if d.Config.Daemon.Addr != "" {
		healthURL := fmt.Sprintf("http://%s/v1/schemas", d.Config.Daemon.Addr)
		client := &http.Client{Timeout: 5 * time.Second}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return info, err
		}

		start := time.Now()
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			info.ResponseTimeMs = time.Since(start).Milliseconds()
			info.Healthy = resp.StatusCode < http.StatusInternalServerError
		}
	}

	return info, nil
}

// PrintDaemonStatus prints the daemon status to stdout.
func PrintDaemonStatus(info *DaemonStatusInfo, asJSON bool) error {
	if asJSON {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if info.Status == "stopped" {
		fmt.Println("Status:  Stopped")
		return nil
	}

	fmt.Println("Status:  Running")
	fmt.Printf("PID:     %d\n", info.PID)

	if len(info.Addresses) > 0 {
		fmt.Println("Listening on:")
		for _, addr := range info.Addresses {
			fmt.Printf("  - %s\n", addr)
		}
	} else {
		fmt.Printf("Address: %s\n", info.Address)
	}

	if info.Healthy {
		fmt.Printf("Health:  Healthy (%dms)\n", info.ResponseTimeMs)
	} else {
		fmt.Println("Health:  Unhealthy")
	}

	return nil
}

// StatusExitCode returns the appropriate exit code for daemon status.
func StatusExitCode(info *DaemonStatusInfo) int {
	if info.Status == "stopped" {
		return 3
	}
	if !info.Healthy {
		return 1
	}
	return 0
}

// Restart restarts the daemon.
func (d *Daemon) Restart(force bool) error {
	pid, _ := d.readPID()
	if pid > 0 && IsProcessRunning(pid) {
		if err := d.Stop(force); err != nil {
			return err
		}
	}

	return d.Start()
}

// TailLogs reads the last N lines from the log file.
func (d *Daemon) TailLogs(lines int) error {
	logPath := d.Config.LogFile()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return errors.Wrap(ErrLogFileNotFound, "path", logPath)
	}

	file, err := os.Open(logPath) // #nosec G304 - path from trusted config
	if err != nil {
		return errors.Wrap(err, "path", logPath)
	}
	defer func() { _ = file.Close() }()

	allLines, err := readAllLines(file)
	if err != nil {
		return err
	}

	printLastNLines(allLines, lines)
	return nil
}

// FollowLogs follows the log file output.
func (d *Daemon) FollowLogs(ctx context.Context, initialLines int) error {
	logPath := d.Config.LogFile()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return errors.Wrap(ErrLogFileNotFound, "path", logPath)
	}

	file, err := os.Open(logPath) // #nosec G304 - path from trusted config
	if err != nil {
		return errors.Wrap(err, "path", logPath)
	}
	defer func() { _ = file.Close() }()

	allLines, err := readAllLines(file)
	if err != nil {
		return err
	}

	printLastNLines(allLines, initialLines)

	currentPos, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.Wrap(err, "path", logPath)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		stat, err := file.Stat()
		if err != nil {
			return errors.Wrap(err, "path", logPath)
		}

		if stat.Size() > currentPos {
			if _, err := file.Seek(currentPos, io.SeekStart); err != nil {
				return errors.Wrap(err, "path", logPath)
			}
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
			currentPos, _ = file.Seek(0, io.SeekCurrent)
		}

		time.Sleep(logFollowPollInterval)
	}
}

func readAllLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func printLastNLines(lines []string, n int) {
	startIdx := 0
	if len(lines) > n {
		startIdx = len(lines) - n
	}
	for _, line := range lines[startIdx:] {
		fmt.Println(line)
	}
}

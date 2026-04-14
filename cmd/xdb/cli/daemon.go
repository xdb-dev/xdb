package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
)

const (
	// daemonChildEnv is set to "1" in the re-execed child process.
	daemonChildEnv = "XDB_DAEMON_CHILD"

	healthCheckTimeout   = 3 * time.Second
	healthCheckPollDelay = 100 * time.Millisecond
	stopTimeout          = 5 * time.Second
	stopPollInterval     = 100 * time.Millisecond
)

func daemonCmd() *cli.Command {
	return &cli.Command{
		Name:               "daemon",
		Usage:              "Manage the XDB daemon (start, stop, status, restart)",
		Category:           "system",
		CustomHelpTemplate: subcommandHelpTemplate,
		Commands: []*cli.Command{
			{
				Name:               "start",
				Usage:              "Start the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "foreground", Usage: "Run in foreground"},
				},
				Action: daemonStartAction,
			},
			{
				Name:               "stop",
				Usage:              "Stop the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonStopAction,
			},
			{
				Name:               "status",
				Usage:              "Show daemon status",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonStatusAction,
			},
			{
				Name:               "restart",
				Usage:              "Restart the daemon",
				CustomHelpTemplate: commandHelpTemplate,
				Action:             daemonRestartAction,
			},
		},
	}
}

// loadAppConfig loads the application config from the --config flag.
func loadAppConfig(cmd *cli.Command) (*Config, error) {
	return LoadConfig(cmd.Root().String("config"))
}

// daemonConfigFrom derives a [daemon.Config] from the application config.
func daemonConfigFrom(cfg *Config) daemon.Config {
	return daemon.Config{
		SocketPath: cfg.SocketPath(),
		LogFile:    cfg.LogFile(),
		Version:    "dev",
	}
}

// isDaemonRunning checks whether the daemon is currently running by reading
// the PID file and verifying the process is alive.
func isDaemonRunning(cfg *Config) bool {
	pid, _ := daemon.ReadPID(cfg.PIDFile())
	return pid > 0 && daemon.IsProcessAlive(pid)
}

func daemonStartAction(ctx context.Context, cmd *cli.Command) error {
	cfg, err := loadAppConfig(cmd)
	if err != nil {
		return err
	}

	// Ensure directory exists.
	if mkErr := os.MkdirAll(cfg.ExpandedDir(), 0o700); mkErr != nil {
		return fmt.Errorf("create xdb directory: %w", mkErr)
	}

	// Already running — report and exit successfully.
	if isDaemonRunning(cfg) {
		pid, _ := daemon.ReadPID(cfg.PIDFile())
		fmt.Fprintf(os.Stderr, "Daemon already running (PID %d)\n", pid)
		return nil
	}

	foreground := cmd.Bool("foreground") || os.Getenv(daemonChildEnv) == "1"
	if foreground {
		return runForeground(ctx, cfg)
	}

	return spawnDaemon(cfg, cmd.Root().String("config"))
}

// runForeground runs the daemon in the current process, blocking until
// the context is canceled or a signal is received.
func runForeground(ctx context.Context, cfg *Config) error {
	s, err := OpenStore(cfg)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = s.Close() }()

	dcfg := daemonConfigFrom(cfg)
	d := daemon.New(dcfg)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "Starting daemon on %s\n", dcfg.SocketPath)

	return d.Start(ctx, s)
}

// spawnDaemon re-execs the current binary as a background child process with
// XDB_DAEMON_CHILD=1 set, redirecting stdout/stderr to the log file. It waits
// for the socket to become available before returning.
func spawnDaemon(cfg *Config, configFlag string) error {
	pidFile := cfg.PIDFile()

	// Already running — nothing to do.
	if isDaemonRunning(cfg) {
		return nil
	}

	// Stale PID file — clean up.
	pid, _ := daemon.ReadPID(pidFile)
	if pid > 0 {
		_ = daemon.RemovePID(pidFile)
	}

	logPath := cfg.LogFile()

	logFile, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o600,
	) // #nosec G304 - path from trusted config
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		_ = logFile.Close()
		return fmt.Errorf("resolve executable: %w", err)
	}

	args := []string{"daemon", "start"}
	if configFlag != "" {
		args = append([]string{"--config", configFlag}, args...)
	}

	child := exec.CommandContext(context.Background(), executable, args...) // #nosec G204 - executable is from os.Executable()
	child.Env = append(os.Environ(), daemonChildEnv+"=1")
	child.Stdout = logFile
	child.Stderr = logFile
	child.Dir = cfg.ExpandedDir()

	// Detach from parent process group so the child outlives us.
	child.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if startErr := child.Start(); startErr != nil {
		_ = logFile.Close()
		return fmt.Errorf("spawn daemon: %w", startErr)
	}

	// Capture the PID before releasing — Release() invalidates the handle and
	// sets Pid to -1 on recent Go versions.
	childPID := child.Process.Pid

	// Release the child so it isn't reaped when we exit.
	_ = child.Process.Release()
	_ = logFile.Close()

	socketPath := cfg.SocketPath()

	// Wait for the socket to accept connections.
	if waitErr := waitForSocket(socketPath, healthCheckTimeout); waitErr != nil {
		return fmt.Errorf("daemon started but not reachable: %w", waitErr)
	}

	fmt.Fprintf(os.Stderr, "Daemon started (PID %d)\n", childPID)
	fmt.Fprintf(os.Stderr, "Listening on %s\n", socketPath)

	return nil
}

// waitForSocket polls until a Unix socket accepts a connection or the timeout
// elapses.
func waitForSocket(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	dialer := net.Dialer{Timeout: 200 * time.Millisecond}

	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(context.Background(), "unix", socketPath)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		time.Sleep(healthCheckPollDelay)
	}

	return fmt.Errorf("timeout waiting for %s", socketPath)
}

func daemonStopAction(_ context.Context, cmd *cli.Command) error {
	cfg, err := loadAppConfig(cmd)
	if err != nil {
		return err
	}

	pidFile := cfg.PIDFile()

	pid, readErr := daemon.ReadPID(pidFile)
	if readErr != nil || pid <= 0 || !daemon.IsProcessAlive(pid) {
		fmt.Fprintln(os.Stderr, "Daemon is not running")

		if pid > 0 {
			_ = daemon.RemovePID(pidFile)
		}

		return nil
	}

	fmt.Fprintf(os.Stderr, "Stopping daemon (PID %d)...\n", pid)

	proc, findErr := os.FindProcess(pid)
	if findErr != nil {
		return fmt.Errorf("find process: %w", findErr)
	}

	if sigErr := proc.Signal(syscall.SIGTERM); sigErr != nil {
		return fmt.Errorf("send SIGTERM: %w", sigErr)
	}

	if !waitForProcessExit(pid, stopTimeout) {
		return fmt.Errorf("daemon did not stop within %s", stopTimeout)
	}

	_ = daemon.RemovePID(pidFile)

	fmt.Fprintln(os.Stderr, "Daemon stopped")

	return nil
}

func daemonStatusAction(_ context.Context, cmd *cli.Command) error {
	cfg, err := loadAppConfig(cmd)
	if err != nil {
		return err
	}

	pidFile := cfg.PIDFile()
	pid, _ := daemon.ReadPID(pidFile)
	status := "stopped"

	if isDaemonRunning(cfg) {
		status = "running"
	}

	result := map[string]string{
		"status": status,
		"socket": cfg.SocketPath(),
	}

	if status == "running" {
		result["pid"] = strconv.Itoa(pid)
	}

	return formatOne(cmd, result)
}

func daemonRestartAction(ctx context.Context, cmd *cli.Command) error {
	// Stop first if running (ignore errors — may not be running).
	_ = daemonStopAction(ctx, cmd)

	return daemonStartAction(ctx, cmd)
}

// waitForProcessExit polls until the process exits or timeout elapses.
func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if !daemon.IsProcessAlive(pid) {
			return true
		}

		time.Sleep(stopPollInterval)
	}

	return !daemon.IsProcessAlive(pid)
}

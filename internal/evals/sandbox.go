package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Sandbox is an isolated xdb installation for one task run. Every command
// runs with a private HOME and with the sandbox bin directory first on PATH.
// The binary is a copy, so the repo path does not appear in the environment
// of the subject.
//
// The root is directly under /tmp. Nested temp paths exceed the Unix socket
// path limit on macOS.
type Sandbox struct {
	Root string
	Home string
	Bin  string
	Work string
	// Fixtures lists the files that the harness copied into Work, sorted.
	Fixtures []string
}

// NewSandbox creates the sandbox layout and copies the binary into it. If
// fixturesDir exists, it copies the fixtures into the work directory.
func NewSandbox(binary, fixturesDir string) (*Sandbox, error) {
	root, err := os.MkdirTemp("/tmp", "xdb-eval-")
	if err != nil {
		return nil, fmt.Errorf("[xdb/evals] create sandbox: %w", err)
	}

	sb := &Sandbox{
		Root: root,
		Home: filepath.Join(root, "home"),
		Bin:  filepath.Join(root, "bin"),
		Work: filepath.Join(root, "work"),
	}

	if err := sb.populate(binary, fixturesDir); err != nil {
		_ = os.RemoveAll(root)

		return nil, err
	}

	return sb, nil
}

func (s *Sandbox) populate(binary, fixturesDir string) error {
	for _, dir := range []string{s.Home, s.Bin, s.Work} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("[xdb/evals] create sandbox dir: %w", err)
		}
	}

	if err := copyFile(binary, filepath.Join(s.Bin, "xdb"), 0o700); err != nil {
		return fmt.Errorf("[xdb/evals] copy binary: %w", err)
	}

	return s.copyFixtures(fixturesDir)
}

func (s *Sandbox) copyFixtures(dir string) error {
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("[xdb/evals] stat fixtures: %w", err)
	}

	if !info.IsDir() {
		return nil
	}

	if copyErr := os.CopyFS(s.Work, os.DirFS(dir)); copyErr != nil {
		return fmt.Errorf("[xdb/evals] copy fixtures: %w", copyErr)
	}

	entries, err := os.ReadDir(s.Work)
	if err != nil {
		return fmt.Errorf("[xdb/evals] list fixtures: %w", err)
	}

	for _, e := range entries {
		s.Fixtures = append(s.Fixtures, e.Name())
	}

	sort.Strings(s.Fixtures)

	return nil
}

// Env returns the process environment with the private HOME of the sandbox
// and its bin directory first on PATH.
func (s *Sandbox) Env() []string {
	env := make([]string, 0, len(os.Environ())+2)

	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") || strings.HasPrefix(kv, "HOME=") {
			continue
		}

		env = append(env, kv)
	}

	return append(env,
		"HOME="+s.Home,
		"PATH="+s.Bin+":"+os.Getenv("PATH"),
	)
}

// Init runs `xdb init` and makes sure that the daemon answers. A task does
// not grade daemon setup.
func (s *Sandbox) Init(ctx context.Context) error {
	out, err := s.Run(ctx, "xdb init")
	if err != nil {
		return err
	}

	if out.Exit != 0 {
		return fmt.Errorf("[xdb/evals] xdb init exited %d: %s", out.Exit, truncate(out.Stderr))
	}

	out, err = s.Run(ctx, "xdb daemon status --quiet")
	if err != nil {
		return err
	}

	if out.Exit != 0 {
		return fmt.Errorf("[xdb/evals] daemon not running after init: %s", truncate(out.Stderr))
	}

	return nil
}

// Run runs a shell command in the work directory with the sandbox
// environment. A non-zero exit code goes into the output. It is not an error.
func (s *Sandbox) Run(ctx context.Context, command string) (CommandOutput, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = s.Work
	cmd.Env = s.Env()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	out := CommandOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	var exitErr *exec.ExitError

	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		out.Exit = exitErr.ExitCode()
	default:
		return out, fmt.Errorf("[xdb/evals] run %q: %w", command, err)
	}

	return out, nil
}

// Context returns the output of `xdb context` from the sandbox binary.
func (s *Sandbox) Context(ctx context.Context) (string, error) {
	out, err := s.Run(ctx, "xdb context")
	if err != nil {
		return "", err
	}

	if out.Exit != 0 {
		return "", fmt.Errorf("[xdb/evals] xdb context exited %d: %s", out.Exit, truncate(out.Stderr))
	}

	return out.Stdout, nil
}

// Close stops the daemon and removes the sandbox. It ignores stop errors,
// because the daemon can already be gone.
func (s *Sandbox) Close() error {
	_, _ = s.Run(context.Background(), "xdb daemon stop")

	if err := os.RemoveAll(s.Root); err != nil {
		return fmt.Errorf("[xdb/evals] remove sandbox: %w", err)
	}

	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()

		return err
	}

	return out.Close()
}

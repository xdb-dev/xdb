package daemon

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// WritePID writes the current process ID to the given path.
func WritePID(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o600)
}

// ReadPID reads a process ID from the given path.
func ReadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// RemovePID removes the PID file at the given path.
func RemovePID(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

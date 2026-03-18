package daemon

import "fmt"

// WritePID writes the current process ID to the given path.
func WritePID(path string) error {
	return fmt.Errorf("daemon: WritePID not implemented")
}

// ReadPID reads a process ID from the given path.
func ReadPID(path string) (int, error) {
	return 0, fmt.Errorf("daemon: ReadPID not implemented")
}

// RemovePID removes the PID file at the given path.
func RemovePID(path string) error {
	return fmt.Errorf("daemon: RemovePID not implemented")
}

package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var ErrAlreadyRunning = errors.New("another instance is already running")

type Lock struct {
	file *os.File
}

// aquire an exclusive, non-blocking lock; rturns an ErrAlreadyRunning error if another process already holds it
func Acquire(name string) (*Lock, error) {
	path, err := lockPath(name)
	if err != nil {
		return nil, fmt.Errorf("lock path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("lock dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("flock: %w", err)
	}

	if f.Truncate(0) == nil {
		if _, err := f.Seek(0, 0); err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
		}
	}

	return &Lock{file: f}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	if cerr := l.file.Close(); err == nil {
		err = cerr
	}
	l.file = nil
	return err
}

func lockPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "hyprtime", name+".lock"), nil
}

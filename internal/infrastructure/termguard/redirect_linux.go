//go:build linux

package termguard

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// redirectStderr points file descriptor 2 at target so that output from
// libraries and child processes that inherit it is captured as well.
func redirectStderr(target *os.File) (func(), error) {
	saved, err := unix.Dup(unix.Stderr)
	if err != nil {
		return nil, fmt.Errorf("duplicate stderr: %w", err)
	}

	if err := unix.Dup3(int(target.Fd()), unix.Stderr, 0); err != nil {
		_ = unix.Close(saved)
		return nil, fmt.Errorf("redirect stderr: %w", err)
	}

	return func() {
		_ = unix.Dup3(saved, unix.Stderr, 0)
		_ = unix.Close(saved)
	}, nil
}

//go:build !darwin && !linux

package termguard

import "os"

// redirectStderr leaves the file descriptor untouched on platforms without
// descriptor duplication; swapping os.Stderr still covers Go callers and the
// child processes they start.
func redirectStderr(_ *os.File) (func(), error) {
	return func() {}, nil
}

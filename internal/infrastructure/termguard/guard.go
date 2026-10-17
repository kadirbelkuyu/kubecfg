// Package termguard keeps stray standard error output away from a terminal
// that a full-screen UI owns. Credential plugins such as gke-gcloud-auth-plugin
// write diagnostics to standard error, which would otherwise be painted over
// the running UI.
package termguard

import (
	"fmt"
	"os"
	"sync"
)

// maxCapturedBytes bounds the captured output; the oldest bytes are dropped.
const maxCapturedBytes = 8 << 10

// Guard holds the standard error redirection for as long as a full-screen UI
// owns the terminal.
type Guard struct {
	original *os.File
	reader   *os.File
	writer   *os.File
	restore  func()
	drained  chan struct{}

	mu       sync.Mutex
	captured []byte
}

// Capture redirects standard error into a pipe and returns the guard that
// restores it. Output written while the guard is active is kept in memory
// instead of reaching the terminal.
func Capture() (*Guard, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	restore, err := redirectStderr(writer)
	if err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, err
	}

	guard := &Guard{
		original: os.Stderr,
		reader:   reader,
		writer:   writer,
		restore:  restore,
		drained:  make(chan struct{}),
	}
	os.Stderr = writer

	go guard.drain()

	return guard, nil
}

// Release restores standard error and returns the output captured meanwhile.
func (g *Guard) Release() string {
	if g == nil {
		return ""
	}

	os.Stderr = g.original
	g.restore()
	_ = g.writer.Close()
	<-g.drained
	_ = g.reader.Close()

	g.mu.Lock()
	defer g.mu.Unlock()

	return string(g.captured)
}

// TerminalStderr returns the standard error the guard took over, so callers
// that hand the terminal to another program can keep writing to it.
func (g *Guard) TerminalStderr() *os.File {
	if g == nil {
		return os.Stderr
	}
	return g.original
}

func (g *Guard) drain() {
	defer close(g.drained)

	buffer := make([]byte, 4096)
	for {
		read, err := g.reader.Read(buffer)
		if read > 0 {
			g.append(buffer[:read])
		}
		if err != nil {
			return
		}
	}
}

func (g *Guard) append(chunk []byte) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.captured = append(g.captured, chunk...)
	if overflow := len(g.captured) - maxCapturedBytes; overflow > 0 {
		g.captured = g.captured[overflow:]
	}
}

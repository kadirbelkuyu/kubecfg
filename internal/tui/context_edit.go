package tui

import (
	"io"
	"os"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

// contextEditSession runs the context definition editor while bubbletea has
// released the terminal. It satisfies tea.ExecCommand.
type contextEditSession struct {
	service        *application.Service
	kubeconfigPath string
	contextName    string
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	terminalStderr *os.File
	update         *application.ContextUpdate
	err            error
}

func (s *contextEditSession) SetStdin(stdin io.Reader) {
	s.stdin = stdin
}

func (s *contextEditSession) SetStdout(stdout io.Writer) {
	s.stdout = stdout
}

func (s *contextEditSession) SetStderr(stderr io.Writer) {
	s.stderr = stderr
}

// Run always reports success so bubbletea restores the terminal; the outcome of
// the edit is read from the session afterwards.
func (s *contextEditSession) Run() error {
	editor := infrastructure.NewExternalEditor(infrastructure.WithEditorStreams(s.streams()))
	s.update, s.err = s.service.EditContextDefinition(s.kubeconfigPath, s.contextName, editor)

	return nil
}

// streams falls back to the process streams for anything bubbletea left unset.
// The editor keeps the terminal standard error so its diagnostics stay visible
// while the TUI holds the captured one.
func (s *contextEditSession) streams() (io.Reader, io.Writer, io.Writer) {
	stdin, stdout := s.stdin, s.stdout
	stderr := s.stderr

	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if s.terminalStderr != nil {
		stderr = s.terminalStderr
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	return stdin, stdout, stderr
}

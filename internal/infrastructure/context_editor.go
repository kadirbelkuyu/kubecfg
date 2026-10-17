package infrastructure

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const editorFilePattern = "kubecfg-context-*.yaml"

// editorEnvVars lists the environment variables consulted for the editor
// command, in order of precedence.
var editorEnvVars = []string{"KUBECFG_EDITOR", "KUBE_EDITOR", "VISUAL", "EDITOR"}

// ExternalEditor stores a context definition in a temporary file and opens it
// with the editor configured through KUBECFG_EDITOR, KUBE_EDITOR, VISUAL, or
// EDITOR.
type ExternalEditor struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type EditorOption func(*ExternalEditor)

// WithEditorStreams wires the editor process to the given terminal streams.
func WithEditorStreams(stdin io.Reader, stdout, stderr io.Writer) EditorOption {
	return func(editor *ExternalEditor) {
		editor.stdin = stdin
		editor.stdout = stdout
		editor.stderr = stderr
	}
}

func NewExternalEditor(options ...EditorOption) *ExternalEditor {
	editor := &ExternalEditor{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	for _, option := range options {
		option(editor)
	}

	return editor
}

// Edit opens the document in the configured editor and returns the saved
// content. The temporary file is removed before Edit returns.
func (e *ExternalEditor) Edit(contextName string, document []byte) ([]byte, error) {
	command, err := resolveEditorCommand()
	if err != nil {
		return nil, err
	}

	path, err := writeEditorFile(document)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(path)
	}()

	if err := e.run(command, path); err != nil {
		return nil, fmt.Errorf("edit context %q: %w", contextName, err)
	}

	edited, err := os.ReadFile(path) // #nosec G304 -- path points at the temporary file created above.
	if err != nil {
		return nil, fmt.Errorf("read edited definition: %w", err)
	}

	if !hasDefinitionContent(edited) {
		return nil, domain.ErrEditAborted
	}

	return edited, nil
}

func (e *ExternalEditor) run(command []string, path string) error {
	arguments := make([]string, 0, len(command))
	arguments = append(arguments, command[1:]...)
	arguments = append(arguments, path)

	// #nosec G204 -- the editor command comes from the user environment and is invoked without a shell.
	process := exec.Command(command[0], arguments...)
	process.Stdin = e.stdin
	process.Stdout = e.stdout
	process.Stderr = e.stderr

	if err := process.Run(); err != nil {
		return fmt.Errorf("run editor %q: %w", strings.Join(command, " "), err)
	}

	return nil
}

func resolveEditorCommand() ([]string, error) {
	for _, name := range editorEnvVars {
		if command := strings.Fields(os.Getenv(name)); len(command) > 0 {
			return command, nil
		}
	}

	fallback := "vi"
	if runtime.GOOS == "windows" {
		fallback = "notepad"
	}

	if _, err := exec.LookPath(fallback); err != nil {
		return nil, fmt.Errorf("no editor found, set EDITOR or KUBECFG_EDITOR")
	}

	return []string{fallback}, nil
}

func writeEditorFile(document []byte) (string, error) {
	file, err := os.CreateTemp("", editorFilePattern)
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}

	// os.CreateTemp creates the file with 0600, so the credentials in the
	// document stay readable only by the current user.
	path := file.Name()
	if _, err := file.Write(document); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write temporary file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close temporary file: %w", err)
	}

	return path, nil
}

// hasDefinitionContent reports whether the document holds anything other than
// comments and blank lines.
func hasDefinitionContent(document []byte) bool {
	for _, line := range strings.Split(string(document), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}

	return false
}

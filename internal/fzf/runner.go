package fzf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrAborted = errors.New("fzf selection aborted")

type Options struct {
	Height  string
	Layout  string
	Prompt  string
	Header  string
	Preview string
}

var command = exec.Command

func Select(items []string, opts Options) (string, error) {
	cmd := command("fzf", argsForOptions(opts)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("open fzf stdin: %w", err)
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start fzf: %w", err)
	}

	go func() {
		defer func() {
			_ = stdin.Close()
		}()

		for _, item := range items {
			_, _ = fmt.Fprintln(stdin, item)
		}
	}()

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 130 {
			return "", ErrAborted
		}

		return "", fmt.Errorf("wait for fzf: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func argsForOptions(opts Options) []string {
	height := opts.Height
	if strings.TrimSpace(height) == "" {
		height = "40%"
	}

	layout := opts.Layout
	if strings.TrimSpace(layout) == "" {
		layout = "reverse"
	}

	prompt := opts.Prompt
	if strings.TrimSpace(prompt) == "" {
		prompt = "context> "
	}

	args := []string{
		"--height=" + height,
		"--layout=" + layout,
		"--prompt=" + prompt,
		"--border",
		"--info=inline",
	}

	if strings.TrimSpace(opts.Header) != "" {
		args = append(args, "--header="+opts.Header)
	}

	if strings.TrimSpace(opts.Preview) != "" {
		args = append(args, "--preview="+opts.Preview)
	}

	return args
}

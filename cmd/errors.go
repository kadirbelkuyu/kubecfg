package cmd

import (
	"errors"
	"fmt"
)

var (
	ErrNameRequired = errors.New("context name is required (use --name flag)")
)

type exitCoder interface {
	ExitCode() int
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

func (e exitError) ExitCode() int {
	return e.code
}

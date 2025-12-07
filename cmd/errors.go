package cmd

import "errors"

var (
	ErrNameRequired = errors.New("context name is required (use --name flag)")
)

package group

import "errors"

var (
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupAlreadyExists = errors.New("group already exists")
	ErrEmptyContextList   = errors.New("group must contain at least one context")
	ErrInvalidGroupName   = errors.New("group name must be alphanumeric with hyphens only")
)

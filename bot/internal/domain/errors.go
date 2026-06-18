package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrLevelLocked    = errors.New("level is locked")
	ErrTopicLocked    = errors.New("topic is locked")
	ErrNotImplemented = errors.New("not implemented")
)

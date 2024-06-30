package cni

import (
	"errors"
)

var (
	ErrCNINotInitialized = errors.New("cni plugin not initialized")
	ErrInvalidConfig     = errors.New("invalid cni config")
	ErrNotFound          = errors.New("not found")
	ErrRead              = errors.New("failed to read config file")
	ErrInvalidResult     = errors.New("invalid result")
	ErrLoad              = errors.New("failed to load cni config")
)

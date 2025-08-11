package task

import "errors"

var (
	ErrNoURLs       = errors.New("no urls provided")
	ErrTaskNotFound = errors.New("task not found")
	ErrTooManyFiles = errors.New("too many files: max 3 per task")
)

func NewErrExtNotAllowed(ext string) error { return errors.New("extension not allowed: " + ext) }

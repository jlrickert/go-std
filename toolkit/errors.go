package toolkit

import "errors"

var (
	ErrNoEnvKey      = errors.New("env key missing")
	ErrEscapeAttempt = errors.New("path escape attempt: operation would access path outside jail")
)

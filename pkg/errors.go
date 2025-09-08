package std

import "errors"

var (
	ErrNoEnvKey = errors.New("env key missing")
)

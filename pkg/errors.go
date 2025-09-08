package std

import "errors"

var (
	ErrNoEnv = errors.New("env key missing")
)

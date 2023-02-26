package process

import (
	"errors"
	"fmt"
)

var (
	ErrProcessNotFound      = errors.New("process not found")
	ErrProcessAlreadyExists = errors.New("process already exists")
)

func processNotFound(id ProcessId) error {
	return fmt.Errorf("%w: %s", ErrProcessNotFound, id)
}

func processExists(id ProcessId) error {
	return fmt.Errorf("%w: %s", ErrProcessAlreadyExists, id)
}

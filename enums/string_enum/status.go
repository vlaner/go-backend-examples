package stringenum

import (
	"errors"
	"fmt"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

var ErrInvalidStatus = errors.New("invalid status")

func (s Status) String() string { return string(s) }

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusActive, StatusSuspended, StatusDeleted:
		return true
	}
	return false
}

func AllStatuses() []Status {
	return []Status{StatusPending, StatusActive, StatusSuspended, StatusDeleted}
}

func ParseStatus(name string) (Status, error) {
	s := Status(name)
	if !s.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, name)
	}
	return s, nil
}

func (s Status) MarshalText() ([]byte, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidStatus, string(s))
	}
	return []byte(s), nil
}

func (s *Status) UnmarshalText(text []byte) error {
	parsed, err := ParseStatus(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

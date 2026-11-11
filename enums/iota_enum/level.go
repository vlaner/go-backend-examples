package iotaenum

import (
	"errors"
	"fmt"
)

type Level uint8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	levelCount
)

var ErrInvalidLevel = errors.New("invalid level")

func levelNames() [levelCount]string {
	return [...]string{
		LevelDebug: "debug",
		LevelInfo:  "info",
		LevelWarn:  "warn",
		LevelError: "error",
	}
}

func (l Level) String() string {
	if !l.Valid() {
		return fmt.Sprintf("level(%d)", l)
	}
	return levelNames()[l]
}

func (l Level) Valid() bool { return l < levelCount }

func Levels() []Level {
	levels := make([]Level, 0, levelCount)
	for l := Level(0); l.Valid(); l++ {
		levels = append(levels, l)
	}
	return levels
}

func ParseLevel(name string) (Level, error) {
	names := levelNames()
	for l := Level(0); l.Valid(); l++ {
		if names[l] == name {
			return l, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrInvalidLevel, name)
}

func (l Level) MarshalText() ([]byte, error) {
	if !l.Valid() {
		return nil, fmt.Errorf("%w: %d", ErrInvalidLevel, l)
	}
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	parsed, err := ParseLevel(string(text))
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

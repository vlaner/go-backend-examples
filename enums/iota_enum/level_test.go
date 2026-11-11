package iotaenum_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	iotaenum "github.com/vlaner/go-backend-examples/enums/iota_enum"
)

func TestLevelsRoundTripThroughText(t *testing.T) {
	t.Parallel()
	levels := iotaenum.Levels()
	if len(levels) == 0 {
		t.Fatal("Levels returned no levels")
	}
	for _, l := range levels {
		if !l.Valid() {
			t.Fatalf("Levels returned invalid level %d", l)
		}
		name := l.String()
		parsed, err := iotaenum.ParseLevel(name)
		if err != nil {
			t.Fatalf("ParseLevel(%q): %v", name, err)
		}
		if parsed != l {
			t.Fatalf("ParseLevel(%q) = %v, want %v", name, parsed, l)
		}
		text, err := l.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%v): %v", l, err)
		}
		var decoded iotaenum.Level
		err = decoded.UnmarshalText(text)
		if err != nil {
			t.Fatalf("UnmarshalText(%q): %v", text, err)
		}
		if decoded != l {
			t.Fatalf("text round trip = %v, want %v", decoded, l)
		}
	}
}

func TestLevelsCoversEveryValidLevel(t *testing.T) {
	t.Parallel()
	levels := iotaenum.Levels()
	for l := iotaenum.Level(0); l.Valid(); l++ {
		if !slices.Contains(levels, l) {
			t.Fatalf("Levels is missing valid level %d", l)
		}
	}
}

func TestParseLevelRejectsUnknown(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", "chatty", "INFO", " info"} {
		_, err := iotaenum.ParseLevel(name)
		if !errors.Is(err, iotaenum.ErrInvalidLevel) {
			t.Fatalf("ParseLevel(%q) error = %v, want ErrInvalidLevel", name, err)
		}
	}
}

func TestInvalidLevelIsRejected(t *testing.T) {
	t.Parallel()
	invalid := iotaenum.Level(200)
	if invalid.Valid() {
		t.Fatal("Level(200) reports itself valid")
	}
	if got, want := invalid.String(), "level(200)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	_, err := invalid.MarshalText()
	if !errors.Is(err, iotaenum.ErrInvalidLevel) {
		t.Fatalf("MarshalText error = %v, want ErrInvalidLevel", err)
	}
}

func TestLevelJSON(t *testing.T) {
	t.Parallel()
	type payload struct {
		Level iotaenum.Level `json:"level"`
	}

	encoded, err := json.Marshal(payload{Level: iotaenum.LevelWarn})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(encoded), `{"level":"warn"}`; got != want {
		t.Fatalf("marshal = %s, want %s", got, want)
	}

	var decoded payload
	err = json.Unmarshal([]byte(`{"level":"error"}`), &decoded)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Level != iotaenum.LevelError {
		t.Fatalf("unmarshal level = %v, want %v", decoded.Level, iotaenum.LevelError)
	}

	err = json.Unmarshal([]byte(`{"level":"chatty"}`), &decoded)
	if !errors.Is(err, iotaenum.ErrInvalidLevel) {
		t.Fatalf("unmarshal unknown error = %v, want ErrInvalidLevel", err)
	}
}

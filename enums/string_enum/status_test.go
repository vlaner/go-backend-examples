package stringenum_test

import (
	"encoding/json"
	"errors"
	"testing"

	stringenum "github.com/vlaner/go-backend-examples/enums/string_enum"
)

func TestStatusValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status stringenum.Status
		want   bool
	}{
		{stringenum.StatusPending, true},
		{stringenum.StatusActive, true},
		{stringenum.StatusSuspended, true},
		{stringenum.StatusDeleted, true},
		{"", false},
		{"Pending", false},
		{"active ", false},
		{"archived", false},
	}
	for _, tc := range tests {
		if got := tc.status.Valid(); got != tc.want {
			t.Fatalf("Status(%q).Valid() = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()
	for _, status := range stringenum.AllStatuses() {
		parsed, err := stringenum.ParseStatus(status.String())
		if err != nil {
			t.Fatalf("ParseStatus(%q): %v", status, err)
		}
		if parsed != status {
			t.Fatalf("ParseStatus(%q) = %q, want %q", status, parsed, status)
		}
	}
	_, err := stringenum.ParseStatus("archived")
	if !errors.Is(err, stringenum.ErrInvalidStatus) {
		t.Fatalf("ParseStatus(archived) error = %v, want ErrInvalidStatus", err)
	}
}

func TestAllStatusesIsACopy(t *testing.T) {
	t.Parallel()
	statuses := stringenum.AllStatuses()
	for i := range statuses {
		statuses[i] = "clobbered"
	}
	for _, status := range stringenum.AllStatuses() {
		if !status.Valid() {
			t.Fatalf("%q stopped being valid after the caller mutated AllStatuses", status)
		}
	}
}

func TestStatusJSON(t *testing.T) {
	t.Parallel()
	type payload struct {
		Status stringenum.Status `json:"status"`
	}

	encoded, err := json.Marshal(payload{Status: stringenum.StatusSuspended})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(encoded), `{"status":"suspended"}`; got != want {
		t.Fatalf("marshal = %s, want %s", got, want)
	}

	var decoded payload
	err = json.Unmarshal([]byte(`{"status":"active"}`), &decoded)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Status != stringenum.StatusActive {
		t.Fatalf("unmarshal status = %q, want %q", decoded.Status, stringenum.StatusActive)
	}

	err = json.Unmarshal([]byte(`{"status":"archived"}`), &decoded)
	if !errors.Is(err, stringenum.ErrInvalidStatus) {
		t.Fatalf("unmarshal unknown error = %v, want ErrInvalidStatus", err)
	}
	_, err = json.Marshal(payload{Status: "archived"})
	if !errors.Is(err, stringenum.ErrInvalidStatus) {
		t.Fatalf("marshal unknown error = %v, want ErrInvalidStatus", err)
	}
}

func TestStatusAsJSONMapKey(t *testing.T) {
	t.Parallel()
	counts := map[stringenum.Status]int{}
	counts[stringenum.StatusActive] = 3
	encoded, err := json.Marshal(counts)
	if err != nil {
		t.Fatalf("marshal map key: %v", err)
	}
	if got, want := string(encoded), `{"active":3}`; got != want {
		t.Fatalf("marshal map key = %s, want %s", got, want)
	}
}

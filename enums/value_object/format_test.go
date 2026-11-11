package valueobject_test

import (
	"encoding/json"
	"errors"
	"testing"

	valueobject "github.com/vlaner/go-backend-examples/enums/value_object"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()
	for _, format := range valueobject.Formats() {
		parsed, err := valueobject.ParseFormat(format.MIME())
		if err != nil {
			t.Fatalf("ParseFormat(%q): %v", format.MIME(), err)
		}
		if parsed != format {
			t.Fatalf("ParseFormat(%q) = %v, want %v", format.MIME(), parsed, format)
		}
	}
	for _, mime := range []string{"", "CSV", "text/csv; charset=utf-8", "application/pdf"} {
		_, err := valueobject.ParseFormat(mime)
		if !errors.Is(err, valueobject.ErrUnknownFormat) {
			t.Fatalf("ParseFormat(%q) error = %v, want ErrUnknownFormat", mime, err)
		}
	}
}

func TestFormatExtension(t *testing.T) {
	t.Parallel()
	tests := []struct {
		format    valueobject.Format
		extension string
	}{
		{valueobject.CSV(), ".csv"},
		{valueobject.JSON(), ".json"},
		{valueobject.XLSX(), ".xlsx"},
	}
	for _, tc := range tests {
		if got := tc.format.Extension(); got != tc.extension {
			t.Fatalf("%v.Extension() = %q, want %q", tc.format, got, tc.extension)
		}
	}
}

func TestFormatComparesAsValue(t *testing.T) {
	t.Parallel()
	parsed, err := valueobject.ParseFormat("text/csv")
	if err != nil {
		t.Fatalf("ParseFormat: %v", err)
	}
	if parsed != valueobject.CSV() {
		t.Fatal("parsed text/csv != valueobject.CSV()")
	}

	renderers := map[valueobject.Format]string{}
	for _, format := range valueobject.Formats() {
		renderers[format] = "render " + format.Extension()
	}
	if got, want := renderers[parsed], "render .csv"; got != want {
		t.Fatalf("renderer lookup by parsed format = %q, want %q", got, want)
	}
}

func TestZeroFormatIsInvalid(t *testing.T) {
	t.Parallel()
	var zero valueobject.Format
	if zero.Valid() {
		t.Fatal("zero Format reports itself valid")
	}
	if zero.MIME() != "" || zero.Extension() != "" {
		t.Fatalf("zero Format = %q/%q, want empty", zero.MIME(), zero.Extension())
	}
	_, err := zero.MarshalText()
	if !errors.Is(err, valueobject.ErrUnknownFormat) {
		t.Fatalf("zero Format marshal error = %v, want ErrUnknownFormat", err)
	}
}

func TestFormatJSON(t *testing.T) {
	t.Parallel()
	type request struct {
		Format valueobject.Format `json:"format"`
	}

	encoded, err := json.Marshal(request{Format: valueobject.CSV()})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(encoded), `{"format":"text/csv"}`; got != want {
		t.Fatalf("marshal = %s, want %s", got, want)
	}

	var decoded request
	err = json.Unmarshal([]byte(`{"format":"application/json"}`), &decoded)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Format != valueobject.JSON() {
		t.Fatalf("unmarshal format = %v, want %v", decoded.Format, valueobject.JSON())
	}

	err = json.Unmarshal([]byte(`{"format":"text/csv; charset=utf-8"}`), &decoded)
	if !errors.Is(err, valueobject.ErrUnknownFormat) {
		t.Fatalf("unmarshal unknown error = %v, want ErrUnknownFormat", err)
	}
}

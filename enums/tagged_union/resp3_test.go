package taggedunion_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	taggedunion "github.com/vlaner/go-backend-examples/enums/tagged_union"
)

func TestSerialize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		val  taggedunion.Value
		want string
	}{
		{"simple_string", taggedunion.NewSimpleString("OK"), "+OK\r\n"},
		{"error", taggedunion.NewError("ERR boom"), "-ERR boom\r\n"},
		{"integer", taggedunion.NewInteger(42), ":42\r\n"},
		{"integer_negative", taggedunion.NewInteger(-7), ":-7\r\n"},
		{"big_number", taggedunion.NewBigNumber("123456789012345678901234567890"), "(123456789012345678901234567890\r\n"},
		{"double", taggedunion.NewDouble(3.14), ",3.14\r\n"},
		{"double_zero", taggedunion.NewDouble(0), ",0\r\n"},
		{"double_int_val", taggedunion.NewDouble(42), ",42\r\n"},
		{"boolean_true", taggedunion.NewBoolean(true), "#t\r\n"},
		{"boolean_false", taggedunion.NewBoolean(false), "#f\r\n"},
		{"null", taggedunion.NewNull(), "_\r\n"},
		{"bulk_string", taggedunion.NewBulkString("hello"), "$5\r\nhello\r\n"},
		{"bulk_string_empty", taggedunion.NewBulkString(""), "$0\r\n\r\n"},
		{"bulk_string_binary", taggedunion.NewBulkString("a\nb"), "$3\r\na\nb\r\n"},
		{"blob_error", taggedunion.NewBlobError("ERR detail"), "!10\r\nERR detail\r\n"},
		{"verbatim", taggedunion.NewVerbatimString("txt", "hello"), "=9\r\ntxt:hello\r\n"},
		{"array", taggedunion.NewArray([]taggedunion.Value{taggedunion.NewInteger(1), taggedunion.NewSimpleString("two")}), "*2\r\n:1\r\n+two\r\n"},
		{"array_empty", taggedunion.NewArray([]taggedunion.Value{}), "*0\r\n"},
		{"set", taggedunion.NewSet([]taggedunion.Value{taggedunion.NewInteger(1)}), "~1\r\n:1\r\n"},
		{"push", taggedunion.NewPush([]taggedunion.Value{taggedunion.NewInteger(1), taggedunion.NewInteger(2)}), ">2\r\n:1\r\n:2\r\n"},
		{"map", taggedunion.NewMap([]taggedunion.KeyValue{{Key: taggedunion.NewBulkString("k"), Value: taggedunion.NewInteger(1)}}), "%1\r\n$1\r\nk\r\n:1\r\n"},
		{"map_empty", taggedunion.NewMap([]taggedunion.KeyValue{}), "%0\r\n"},
		{"attribute", taggedunion.NewAttribute([]taggedunion.KeyValue{{Key: taggedunion.NewBulkString("a"), Value: taggedunion.NewInteger(1)}}, taggedunion.NewSimpleString("OK")), "|1\r\n$1\r\na\r\n:1\r\n+OK\r\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := taggedunion.Serialize(tc.val)
			if got != tc.want {
				t.Errorf("Serialize:\nwant %q\ngot  %q", tc.want, got)
			}
			parsed, err := taggedunion.Parse(tc.want)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !reflect.DeepEqual(parsed, tc.val) {
				t.Errorf("Parse:\nwant %#v\ngot  %#v", tc.val, parsed)
			}
		})
	}
}

func TestSerializeDoubleSpecial(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		f    float64
		want string
	}{
		{"inf", math.Inf(1), ",inf\r\n"},
		{"neg_inf", math.Inf(-1), ",-inf\r\n"},
		{"nan", math.NaN(), ",nan\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := taggedunion.Serialize(taggedunion.NewDouble(tc.f))
			if got != tc.want {
				t.Fatalf("Serialize: want %q, got %q", tc.want, got)
			}
			parsed, err := taggedunion.Parse(tc.want)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if parsed.Kind != taggedunion.KindDouble {
				t.Fatalf("parsed kind = %v, want double", parsed.Kind)
			}
			switch tc.name {
			case "inf":
				if !math.IsInf(parsed.Float, 1) {
					t.Errorf("want +inf, got %v", parsed.Float)
				}
			case "neg_inf":
				if !math.IsInf(parsed.Float, -1) {
					t.Errorf("want -inf, got %v", parsed.Float)
				}
			case "nan":
				if !math.IsNaN(parsed.Float) {
					t.Errorf("want nan, got %v", parsed.Float)
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty", "", taggedunion.ErrIncomplete},
		{"truncated_int", ":12", taggedunion.ErrIncomplete},
		{"unknown", "x\r\n", nil},
		{"bad_int", ":abc\r\n", nil},
		{"bad_bool", "#x\r\n", nil},
		{"bad_double", ",abc\r\n", nil},
		{"bad_verbatim", "=3\r\nabc\r\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := taggedunion.Parse(tc.in)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()
	if taggedunion.KindSimpleString.String() != "simple_string" {
		t.Errorf("got %q", taggedunion.KindSimpleString.String())
	}
	if taggedunion.Kind(99).String() != "unknown" {
		t.Errorf("got %q", taggedunion.Kind(99).String())
	}
}

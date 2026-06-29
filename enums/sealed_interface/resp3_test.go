package sealedinterface_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	sealedinterface "github.com/vlaner/go-backend-examples/enums/sealed_interface"
)

func TestSerialize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		val  sealedinterface.Type
		want string
	}{
		{"simple_string", sealedinterface.SimpleString{Value: "OK"}, "+OK\r\n"},
		{"error", sealedinterface.ErrorReply{Value: "ERR boom"}, "-ERR boom\r\n"},
		{"integer", sealedinterface.Integer{Value: 42}, ":42\r\n"},
		{"integer_negative", sealedinterface.Integer{Value: -7}, ":-7\r\n"},
		{"big_number", sealedinterface.BigNumber{Value: "123456789012345678901234567890"}, "(123456789012345678901234567890\r\n"},
		{"double", sealedinterface.Double{Value: 3.14}, ",3.14\r\n"},
		{"double_zero", sealedinterface.Double{Value: 0}, ",0\r\n"},
		{"double_int_val", sealedinterface.Double{Value: 42}, ",42\r\n"},
		{"boolean_true", sealedinterface.Boolean{Value: true}, "#t\r\n"},
		{"boolean_false", sealedinterface.Boolean{Value: false}, "#f\r\n"},
		{"null", sealedinterface.Null{}, "_\r\n"},
		{"bulk_string", sealedinterface.BulkString{Value: "hello"}, "$5\r\nhello\r\n"},
		{"bulk_string_empty", sealedinterface.BulkString{Value: ""}, "$0\r\n\r\n"},
		{"bulk_string_binary", sealedinterface.BulkString{Value: "a\nb"}, "$3\r\na\nb\r\n"},
		{"blob_error", sealedinterface.BlobError{Value: "ERR detail"}, "!10\r\nERR detail\r\n"},
		{"verbatim", sealedinterface.VerbatimString{Format: "txt", Value: "hello"}, "=9\r\ntxt:hello\r\n"},
		{"array", sealedinterface.Array{Items: []sealedinterface.Type{sealedinterface.Integer{Value: 1}, sealedinterface.SimpleString{Value: "two"}}}, "*2\r\n:1\r\n+two\r\n"},
		{"array_empty", sealedinterface.Array{Items: []sealedinterface.Type{}}, "*0\r\n"},
		{"set", sealedinterface.Set{Items: []sealedinterface.Type{sealedinterface.Integer{Value: 1}}}, "~1\r\n:1\r\n"},
		{"push", sealedinterface.Push{Items: []sealedinterface.Type{sealedinterface.Integer{Value: 1}, sealedinterface.Integer{Value: 2}}}, ">2\r\n:1\r\n:2\r\n"},
		{"map", sealedinterface.Map{Pairs: []sealedinterface.KeyValue{{Key: sealedinterface.BulkString{Value: "k"}, Value: sealedinterface.Integer{Value: 1}}}}, "%1\r\n$1\r\nk\r\n:1\r\n"},
		{"map_empty", sealedinterface.Map{Pairs: []sealedinterface.KeyValue{}}, "%0\r\n"},
		{"attribute", sealedinterface.Attribute{Pairs: []sealedinterface.KeyValue{{Key: sealedinterface.BulkString{Value: "a"}, Value: sealedinterface.Integer{Value: 1}}}, Value: sealedinterface.SimpleString{Value: "OK"}}, "|1\r\n$1\r\na\r\n:1\r\n+OK\r\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.val.Serialize()
			if got != tc.want {
				t.Errorf("Serialize:\nwant %q\ngot  %q", tc.want, got)
			}
			parsed, err := sealedinterface.Parse(tc.want)
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
			got := (sealedinterface.Double{Value: tc.f}).Serialize()
			if got != tc.want {
				t.Fatalf("Serialize: want %q, got %q", tc.want, got)
			}
			parsed, err := sealedinterface.Parse(tc.want)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			d, ok := parsed.(sealedinterface.Double)
			if !ok {
				t.Fatalf("parsed type = %T, want Double", parsed)
			}
			switch tc.name {
			case "inf":
				if !math.IsInf(d.Value, 1) {
					t.Errorf("want +inf, got %v", d.Value)
				}
			case "neg_inf":
				if !math.IsInf(d.Value, -1) {
					t.Errorf("want -inf, got %v", d.Value)
				}
			case "nan":
				if !math.IsNaN(d.Value) {
					t.Errorf("want nan, got %v", d.Value)
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
		{"empty", "", sealedinterface.ErrIncomplete},
		{"truncated_int", ":12", sealedinterface.ErrIncomplete},
		{"unknown", "x\r\n", nil},
		{"bad_int", ":abc\r\n", nil},
		{"bad_bool", "#x\r\n", nil},
		{"bad_double", ",abc\r\n", nil},
		{"bad_verbatim", "=3\r\nabc\r\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := sealedinterface.Parse(tc.in)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestTypeSealed(t *testing.T) {
	t.Parallel()
	var v sealedinterface.Type = sealedinterface.Null{}
	if v.Serialize() != "_\r\n" {
		t.Fatal("Null does not satisfy Type")
	}
}

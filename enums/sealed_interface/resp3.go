package sealedinterface

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Type interface {
	isType()
	Serialize() string
}

type KeyValue struct {
	Key   Type
	Value Type
}

type (
	SimpleString   struct{ Value string }
	ErrorReply     struct{ Value string }
	Integer        struct{ Value int64 }
	BigNumber      struct{ Value string }
	Double         struct{ Value float64 }
	Boolean        struct{ Value bool }
	Null           struct{}
	BulkString     struct{ Value string }
	BlobError      struct{ Value string }
	VerbatimString struct {
		Format string
		Value  string
	}
)

type (
	Array     struct{ Items []Type }
	Set       struct{ Items []Type }
	Push      struct{ Items []Type }
	Map       struct{ Pairs []KeyValue }
	Attribute struct {
		Pairs []KeyValue
		Value Type
	}
)

func (SimpleString) isType()   {}
func (ErrorReply) isType()     {}
func (Integer) isType()        {}
func (BigNumber) isType()      {}
func (Double) isType()         {}
func (Boolean) isType()        {}
func (Null) isType()           {}
func (BulkString) isType()     {}
func (BlobError) isType()      {}
func (VerbatimString) isType() {}
func (Array) isType()          {}
func (Set) isType()            {}
func (Push) isType()           {}
func (Map) isType()            {}
func (Attribute) isType()      {}

func (s SimpleString) Serialize() string { return "+" + s.Value + "\r\n" }
func (e ErrorReply) Serialize() string   { return "-" + e.Value + "\r\n" }
func (n Integer) Serialize() string      { return ":" + strconv.FormatInt(n.Value, 10) + "\r\n" }
func (n BigNumber) Serialize() string    { return "(" + n.Value + "\r\n" }
func (d Double) Serialize() string       { return "," + formatDouble(d.Value) + "\r\n" }
func (b Boolean) Serialize() string {
	if b.Value {
		return "#t\r\n"
	}
	return "#f\r\n"
}
func (Null) Serialize() string         { return "_\r\n" }
func (b BulkString) Serialize() string { return serializeBlob('$', b.Value) }
func (e BlobError) Serialize() string  { return serializeBlob('!', e.Value) }
func (v VerbatimString) Serialize() string {
	return serializeBlob('=', v.Format+":"+v.Value)
}
func (a Array) Serialize() string { return serializeAggregate('*', a.Items) }
func (s Set) Serialize() string   { return serializeAggregate('~', s.Items) }
func (p Push) Serialize() string  { return serializeAggregate('>', p.Items) }
func (m Map) Serialize() string   { return serializePairs('%', m.Pairs) }
func (a Attribute) Serialize() string {
	out := serializePairs('|', a.Pairs)
	if a.Value != nil {
		out += a.Value.Serialize()
	}
	return out
}

func serializeBlob(marker byte, s string) string {
	var b strings.Builder
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteString("\r\n")
	b.WriteString(s)
	b.WriteString("\r\n")
	return b.String()
}

func serializeAggregate(marker byte, items []Type) string {
	var b strings.Builder
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(items)))
	b.WriteString("\r\n")
	for _, item := range items {
		b.WriteString(item.Serialize())
	}
	return b.String()
}

func serializePairs(marker byte, pairs []KeyValue) string {
	var b strings.Builder
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(pairs)))
	b.WriteString("\r\n")
	for _, kv := range pairs {
		b.WriteString(kv.Key.Serialize())
		b.WriteString(kv.Value.Serialize())
	}
	return b.String()
}

func formatDouble(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

var ErrIncomplete = errors.New("resp3: incomplete data")

func Parse(s string) (Type, error) {
	p := &parser{data: []byte(s)}
	return p.parseValue()
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) parseValue() (Type, error) {
	if p.pos >= len(p.data) {
		return nil, ErrIncomplete
	}
	marker := p.data[p.pos]
	p.pos++
	switch marker {
	case '+':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		return SimpleString{Value: line}, nil
	case '-':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		return ErrorReply{Value: line}, nil
	case ':':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("resp3: invalid integer %q: %w", line, err)
		}
		return Integer{Value: n}, nil
	case '(':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		return BigNumber{Value: line}, nil
	case ',':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		f, err := parseDouble(line)
		if err != nil {
			return nil, err
		}
		return Double{Value: f}, nil
	case '#':
		line, err := p.readLine()
		if err != nil {
			return nil, err
		}
		switch line {
		case "t":
			return Boolean{Value: true}, nil
		case "f":
			return Boolean{Value: false}, nil
		default:
			return nil, fmt.Errorf("resp3: invalid boolean %q", line)
		}
	case '_':
		_, err := p.readLine()
		if err != nil {
			return nil, err
		}
		return Null{}, nil
	case '$':
		s, err := p.parseBlob()
		if err != nil {
			return nil, err
		}
		return BulkString{Value: s}, nil
	case '!':
		s, err := p.parseBlob()
		if err != nil {
			return nil, err
		}
		return BlobError{Value: s}, nil
	case '=':
		return p.parseVerbatimString()
	case '*':
		items, err := p.parseItems()
		if err != nil {
			return nil, err
		}
		return Array{Items: items}, nil
	case '~':
		items, err := p.parseItems()
		if err != nil {
			return nil, err
		}
		return Set{Items: items}, nil
	case '>':
		items, err := p.parseItems()
		if err != nil {
			return nil, err
		}
		return Push{Items: items}, nil
	case '%':
		pairs, err := p.parsePairs()
		if err != nil {
			return nil, err
		}
		return Map{Pairs: pairs}, nil
	case '|':
		pairs, err := p.parsePairs()
		if err != nil {
			return nil, err
		}
		inner, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		return Attribute{Pairs: pairs, Value: inner}, nil
	default:
		return nil, fmt.Errorf("resp3: unknown type byte %q", string(marker))
	}
}

func (p *parser) readLine() (string, error) {
	start := p.pos
	idx := bytes.IndexByte(p.data[start:], '\n')
	if idx < 0 {
		return "", ErrIncomplete
	}
	end := start + idx
	line := p.data[start:end]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	p.pos = end + 1
	return string(line), nil
}

func (p *parser) readCount() (int, error) {
	line, err := p.readLine()
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("resp3: invalid length %q: %w", line, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("resp3: negative length %d", n)
	}
	return n, nil
}

func (p *parser) readBytes(n int) ([]byte, error) {
	if p.pos+n > len(p.data) {
		return nil, ErrIncomplete
	}
	b := p.data[p.pos : p.pos+n]
	p.pos += n
	err := p.consumeCRLF()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (p *parser) consumeCRLF() error {
	if p.pos+2 > len(p.data) || p.data[p.pos] != '\r' || p.data[p.pos+1] != '\n' {
		return ErrIncomplete
	}
	p.pos += 2
	return nil
}

func (p *parser) parseBlob() (string, error) {
	n, err := p.readCount()
	if err != nil {
		return "", err
	}
	payload, err := p.readBytes(n)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (p *parser) parseVerbatimString() (VerbatimString, error) {
	n, err := p.readCount()
	if err != nil {
		return VerbatimString{}, err
	}
	payload, err := p.readBytes(n)
	if err != nil {
		return VerbatimString{}, err
	}
	format, value, found := strings.Cut(string(payload), ":")
	if !found {
		return VerbatimString{}, fmt.Errorf("resp3: invalid verbatim string %q", payload)
	}
	return VerbatimString{Format: format, Value: value}, nil
}

func (p *parser) parseItems() ([]Type, error) {
	n, err := p.readCount()
	if err != nil {
		return nil, err
	}
	items := make([]Type, 0, n)
	for range n {
		item, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (p *parser) parsePairs() ([]KeyValue, error) {
	n, err := p.readCount()
	if err != nil {
		return nil, err
	}
	pairs := make([]KeyValue, 0, n)
	for range n {
		k, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, KeyValue{Key: k, Value: v})
	}
	return pairs, nil
}

func parseDouble(s string) (float64, error) {
	switch s {
	case "inf":
		return math.Inf(1), nil
	case "-inf":
		return math.Inf(-1), nil
	case "nan":
		return math.NaN(), nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("resp3: invalid double %q: %w", s, err)
	}
	return f, nil
}

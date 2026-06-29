package taggedunion

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Kind uint8

const (
	KindSimpleString Kind = iota
	KindError
	KindInteger
	KindBigNumber
	KindDouble
	KindBoolean
	KindNull
	KindBulkString
	KindBlobError
	KindVerbatimString
	KindArray
	KindSet
	KindPush
	KindMap
	KindAttribute
)

func (k Kind) String() string {
	switch k {
	case KindSimpleString:
		return "simple_string"
	case KindError:
		return "error"
	case KindInteger:
		return "integer"
	case KindBigNumber:
		return "big_number"
	case KindDouble:
		return "double"
	case KindBoolean:
		return "boolean"
	case KindNull:
		return "null"
	case KindBulkString:
		return "bulk_string"
	case KindBlobError:
		return "blob_error"
	case KindVerbatimString:
		return "verbatim_string"
	case KindArray:
		return "array"
	case KindSet:
		return "set"
	case KindPush:
		return "push"
	case KindMap:
		return "map"
	case KindAttribute:
		return "attribute"
	default:
		return "unknown"
	}
}

type KeyValue struct {
	Key   Value
	Value Value
}

type Value struct {
	Kind   Kind
	Str    string
	Int    int64
	Float  float64
	Bool   bool
	Format string
	Items  []Value
	Pairs  []KeyValue
	Inner  *Value
}

func NewSimpleString(s string) Value { return Value{Kind: KindSimpleString, Str: s} }
func NewError(s string) Value        { return Value{Kind: KindError, Str: s} }
func NewBigNumber(s string) Value    { return Value{Kind: KindBigNumber, Str: s} }
func NewBulkString(s string) Value   { return Value{Kind: KindBulkString, Str: s} }
func NewBlobError(s string) Value    { return Value{Kind: KindBlobError, Str: s} }
func NewVerbatimString(format, s string) Value {
	return Value{Kind: KindVerbatimString, Format: format, Str: s}
}
func NewInteger(n int64) Value      { return Value{Kind: KindInteger, Int: n} }
func NewDouble(f float64) Value     { return Value{Kind: KindDouble, Float: f} }
func NewBoolean(b bool) Value       { return Value{Kind: KindBoolean, Bool: b} }
func NewNull() Value                { return Value{Kind: KindNull} }
func NewArray(items []Value) Value  { return Value{Kind: KindArray, Items: items} }
func NewSet(items []Value) Value    { return Value{Kind: KindSet, Items: items} }
func NewPush(items []Value) Value   { return Value{Kind: KindPush, Items: items} }
func NewMap(pairs []KeyValue) Value { return Value{Kind: KindMap, Pairs: pairs} }
func NewAttribute(pairs []KeyValue, inner Value) Value {
	return Value{Kind: KindAttribute, Pairs: pairs, Inner: &inner}
}

func Serialize(v Value) string {
	var b strings.Builder
	serialize(&b, v)
	return b.String()
}

func serialize(b *strings.Builder, v Value) {
	switch v.Kind {
	case KindSimpleString:
		b.WriteByte('+')
		b.WriteString(v.Str)
		b.WriteString("\r\n")
	case KindError:
		b.WriteByte('-')
		b.WriteString(v.Str)
		b.WriteString("\r\n")
	case KindInteger:
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(v.Int, 10))
		b.WriteString("\r\n")
	case KindBigNumber:
		b.WriteByte('(')
		b.WriteString(v.Str)
		b.WriteString("\r\n")
	case KindDouble:
		b.WriteByte(',')
		b.WriteString(formatDouble(v.Float))
		b.WriteString("\r\n")
	case KindBoolean:
		b.WriteByte('#')
		if v.Bool {
			b.WriteByte('t')
		} else {
			b.WriteByte('f')
		}
		b.WriteString("\r\n")
	case KindNull:
		b.WriteString("_\r\n")
	case KindBulkString:
		writeBlob(b, '$', v.Str)
	case KindBlobError:
		writeBlob(b, '!', v.Str)
	case KindVerbatimString:
		writeBlob(b, '=', v.Format+":"+v.Str)
	case KindArray:
		writeAggregate(b, '*', v.Items)
	case KindSet:
		writeAggregate(b, '~', v.Items)
	case KindPush:
		writeAggregate(b, '>', v.Items)
	case KindMap:
		writePairs(b, '%', v.Pairs)
	case KindAttribute:
		writePairs(b, '|', v.Pairs)
		if v.Inner != nil {
			serialize(b, *v.Inner)
		}
	}
}

func writeBlob(b *strings.Builder, marker byte, s string) {
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteString("\r\n")
	b.WriteString(s)
	b.WriteString("\r\n")
}

func writeAggregate(b *strings.Builder, marker byte, items []Value) {
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(items)))
	b.WriteString("\r\n")
	for _, item := range items {
		serialize(b, item)
	}
}

func writePairs(b *strings.Builder, marker byte, pairs []KeyValue) {
	b.WriteByte(marker)
	b.WriteString(strconv.Itoa(len(pairs)))
	b.WriteString("\r\n")
	for _, kv := range pairs {
		serialize(b, kv.Key)
		serialize(b, kv.Value)
	}
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

func Parse(s string) (Value, error) {
	p := &parser{data: []byte(s)}
	return p.parseValue()
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) parseValue() (Value, error) {
	if p.pos >= len(p.data) {
		return Value{}, ErrIncomplete
	}
	marker := p.data[p.pos]
	p.pos++
	switch marker {
	case '+':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		return NewSimpleString(line), nil
	case '-':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		return NewError(line), nil
	case ':':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("resp3: invalid integer %q: %w", line, err)
		}
		return NewInteger(n), nil
	case '(':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		return NewBigNumber(line), nil
	case ',':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		f, err := parseDouble(line)
		if err != nil {
			return Value{}, err
		}
		return NewDouble(f), nil
	case '#':
		line, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		switch line {
		case "t":
			return NewBoolean(true), nil
		case "f":
			return NewBoolean(false), nil
		default:
			return Value{}, fmt.Errorf("resp3: invalid boolean %q", line)
		}
	case '_':
		_, err := p.readLine()
		if err != nil {
			return Value{}, err
		}
		return NewNull(), nil
	case '$':
		return p.parseBlob(KindBulkString)
	case '!':
		return p.parseBlob(KindBlobError)
	case '=':
		return p.parseVerbatim()
	case '*':
		return p.parseAggregate(KindArray)
	case '~':
		return p.parseAggregate(KindSet)
	case '>':
		return p.parseAggregate(KindPush)
	case '%':
		return p.parseMap()
	case '|':
		return p.parseAttribute()
	default:
		return Value{}, fmt.Errorf("resp3: unknown type byte %q", string(marker))
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

func (p *parser) parseBlob(kind Kind) (Value, error) {
	n, err := p.readCount()
	if err != nil {
		return Value{}, err
	}
	payload, err := p.readBytes(n)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: kind, Str: string(payload)}, nil
}

func (p *parser) parseVerbatim() (Value, error) {
	n, err := p.readCount()
	if err != nil {
		return Value{}, err
	}
	payload, err := p.readBytes(n)
	if err != nil {
		return Value{}, err
	}
	format, text, found := strings.Cut(string(payload), ":")
	if !found {
		return Value{}, fmt.Errorf("resp3: invalid verbatim string %q", payload)
	}
	return NewVerbatimString(format, text), nil
}

func (p *parser) parseAggregate(kind Kind) (Value, error) {
	n, err := p.readCount()
	if err != nil {
		return Value{}, err
	}
	items := make([]Value, 0, n)
	for range n {
		item, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		items = append(items, item)
	}
	return Value{Kind: kind, Items: items}, nil
}

func (p *parser) readPairs(n int) ([]KeyValue, error) {
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

func (p *parser) parseMap() (Value, error) {
	n, err := p.readCount()
	if err != nil {
		return Value{}, err
	}
	pairs, err := p.readPairs(n)
	if err != nil {
		return Value{}, err
	}
	return NewMap(pairs), nil
}

func (p *parser) parseAttribute() (Value, error) {
	n, err := p.readCount()
	if err != nil {
		return Value{}, err
	}
	pairs, err := p.readPairs(n)
	if err != nil {
		return Value{}, err
	}
	inner, err := p.parseValue()
	if err != nil {
		return Value{}, err
	}
	return NewAttribute(pairs, inner), nil
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

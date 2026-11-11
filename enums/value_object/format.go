package valueobject

import (
	"errors"
	"fmt"
)

type Format struct {
	mime      string
	extension string
}

var ErrUnknownFormat = errors.New("unknown format")

func CSV() Format { return Format{mime: "text/csv", extension: ".csv"} }

func JSON() Format { return Format{mime: "application/json", extension: ".json"} }

func XLSX() Format {
	return Format{mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", extension: ".xlsx"}
}

func Formats() []Format { return []Format{CSV(), JSON(), XLSX()} }

func (f Format) MIME() string { return f.mime }

func (f Format) Extension() string { return f.extension }

func (f Format) String() string { return f.mime }

func (f Format) Valid() bool {
	switch f {
	case CSV(), JSON(), XLSX():
		return true
	}
	return false
}

func ParseFormat(mime string) (Format, error) {
	for _, format := range Formats() {
		if format.mime == mime {
			return format, nil
		}
	}
	return Format{}, fmt.Errorf("%w: %q", ErrUnknownFormat, mime)
}

func (f Format) MarshalText() ([]byte, error) {
	if !f.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, f.mime)
	}
	return []byte(f.mime), nil
}

func (f *Format) UnmarshalText(text []byte) error {
	parsed, err := ParseFormat(string(text))
	if err != nil {
		return err
	}
	*f = parsed
	return nil
}

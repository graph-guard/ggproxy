package metadata

import (
	"bytes"
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

type Metadata struct {
	Name string
	Tags []string
}

func Split(s []byte) (header, body []byte, err error) {
	si := consumeEmptyLines(s)

	if len(si) < 1 {
		return nil, s, nil
	}

	if !bytes.HasPrefix(si, delimiter) {
		return nil, s, nil
	}
	si = si[len(delimiter):]

	var ok bool
	if si, ok = consumeAfterDelimiter(si); !ok {
		return nil, s, ErrExpectedDelimiter
	}

	if len(si) < 1 {
		return nil, s, ErrExpectedDelimiter
	}

	header, body, ok = consumeHeader(si)
	if !ok {
		return nil, s, ErrExpectedDelimiter
	}

	return header, body, nil
}

func Parse(s []byte) (m Metadata, body []byte, err error) {
	header, body, err := Split(s)
	if err != nil {
		return m, nil, err
	} else if header == nil {
		return m, body, nil
	}

	d := yaml.NewDecoder(bytes.NewReader(header))
	d.KnownFields(true)
	if err := d.Decode(&m); err != nil {
		return m, body, fmt.Errorf("decoding yaml: %w", err)
	}

	return m, body, err
}

var delimiter = []byte("---")
var ErrExpectedDelimiter = errors.New("expected delimiter")

func consumeAfterDelimiter(s []byte) (after []byte, ok bool) {
	for i := range s {
		if s[i] == '\n' {
			return s[i+1:], true
		}
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\r' {
			continue
		}
		return nil, false
	}
	return nil, true
}

func consumeHeader(s []byte) (header, after []byte, ok bool) {
	for x := s; len(x) > 0; {
		if x[0] == '\n' {
			x = x[1:]
			if bytes.HasPrefix(x, delimiter) {
				header = s[:len(s)-len(x)]
				x = x[len(delimiter):]
				next, ok := consumeAfterDelimiter(x)
				if !ok {
					return nil, nil, false
				}
				return header, next, true
			}
			continue
		}
		x = x[1:]
	}
	return s, nil, true
}

func consumeEmptyLines(s []byte) []byte {
	li := 0
	// Find first non-empty line
	for i := li; i < len(s); i++ {
		if s[i] == '\n' {
			li = i + 1
		}
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\r' {
			continue
		}
		break
	}
	return s[li:]
}

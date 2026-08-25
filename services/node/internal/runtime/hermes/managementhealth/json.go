package managementhealth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	ErrMalformedResponse = errors.New("malformed Hermes response")
	ErrResponseTooLarge  = errors.New("Hermes response exceeds the fixed size limit")
)

func decodeStrict(data []byte, maxBytes int, dst any) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: empty body", ErrMalformedResponse)
	}
	if len(data) > maxBytes {
		return ErrResponseTooLarge
	}
	if err := rejectDuplicateJSONFields(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: invalid JSON shape", ErrMalformedResponse)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing JSON value", ErrMalformedResponse)
	}
	return nil
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var scanValue func() error
	scanValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("%w: invalid JSON", ErrMalformedResponse)
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return fmt.Errorf("%w: invalid object key", ErrMalformedResponse)
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("%w: invalid object key", ErrMalformedResponse)
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("%w: duplicate JSON field", ErrMalformedResponse)
				}
				seen[key] = struct{}{}
				if err := scanValue(); err != nil {
					return err
				}
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("%w: invalid object", ErrMalformedResponse)
			}
		case '[':
			for decoder.More() {
				if err := scanValue(); err != nil {
					return err
				}
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("%w: invalid array", ErrMalformedResponse)
			}
		default:
			return fmt.Errorf("%w: invalid JSON delimiter", ErrMalformedResponse)
		}
		return nil
	}
	if err := scanValue(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing JSON value", ErrMalformedResponse)
	}
	return nil
}
